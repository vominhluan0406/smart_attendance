package service

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/repository"
	"github.com/smart-attendance/smart-attendance/internal/timezone"
	"gorm.io/gorm"
)

var (
	ErrTOTPInvalid    = errors.New("invalid QR/TOTP code")
	ErrIPNotAllowed   = errors.New("IP address not in whitelist")
	ErrLocationOutside = errors.New("location outside allowed area")
	ErrNoBranch       = errors.New("user not assigned to any branch")
	ErrMethodRequired = errors.New("at least one verification method must pass")
)

type AttendanceService struct {
	attendanceRepo *repository.AttendanceRepository
	logRepo        *repository.AttendanceLogRepository
	shiftRepo      *repository.ShiftRepository
	branchService  *BranchService
	userService    *UserService
	totpService    *TOTPService
	ipValidator    *IPValidator
	locValidator   *LocationValidator
	leaveRepo      *repository.LeaveRepository
	antiFraud      *AntiFraudService
}

func NewAttendanceService(
	attendanceRepo *repository.AttendanceRepository,
	logRepo *repository.AttendanceLogRepository,
	shiftRepo *repository.ShiftRepository,
	branchService *BranchService,
	userService *UserService,
	totpService *TOTPService,
	ipValidator *IPValidator,
	locValidator *LocationValidator,
	leaveRepo *repository.LeaveRepository,
	antiFraud *AntiFraudService,
) *AttendanceService {
	return &AttendanceService{
		attendanceRepo: attendanceRepo,
		logRepo:        logRepo,
		shiftRepo:      shiftRepo,
		branchService:  branchService,
		userService:    userService,
		totpService:    totpService,
		ipValidator:    ipValidator,
		locValidator:   locValidator,
		leaveRepo:      leaveRepo,
		antiFraud:      antiFraud,
	}
}

// LogTimeInput is the input for each QR scan / time log.
type LogTimeInput struct {
	UserID            string   `json:"user_id"`
	TOTPCode          string   `json:"totp_code"`
	ScannedBranchID   string   `json:"scanned_branch_id"`
	Lat               *float64 `json:"lat"`
	Lng               *float64 `json:"lng"`
	AccuracyM         *float64 `json:"accuracy_m"`
	IP                string   `json:"ip"`
	UserAgent         string   `json:"user_agent"`
	DeviceFingerprint string   `json:"device_fingerprint"`
	FaceVerified      bool     `json:"face_verified"`
	NFCVerified       bool     `json:"nfc_verified"`
	PasswordVerified  bool     `json:"password_verified"`
	BiometricVerified bool     `json:"biometric_verified"`
}

// LogTimeResult is returned after a successful time log.
type LogTimeResult struct {
	Log              *models.AttendanceLog `json:"log"`
	Attendance       *models.Attendance    `json:"attendance"`
	TOTPVerified     bool                  `json:"totp_verified"`
	IPVerified       bool                  `json:"ip_verified"`
	LocVerified      bool                  `json:"loc_verified"`
	FaceVerified     bool                  `json:"face_verified"`
	NFCVerified      bool                  `json:"nfc_verified"`
	PasswordVerified bool                  `json:"password_verified"`
	WiFiGPSVerified  bool                  `json:"wifi_gps_verified"`
	LogCount         int                   `json:"log_count"`
	NewDevice        bool                  `json:"new_device"`
	AnomalyFlag      bool                  `json:"anomaly_flag"`
	AnomalyScore     float64               `json:"anomaly_score"`
}

// LogTime records a time scan (replaces separate CheckIn/CheckOut).
// Each scan creates an AttendanceLog and updates the daily Attendance summary.
func (s *AttendanceService) LogTime(input LogTimeInput) (*LogTimeResult, error) {
	// 1. Get user and branch
	user, err := s.userService.GetByID(input.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	if user.BranchID == nil {
		return nil, ErrNoBranch
	}

	branch, err := s.branchService.GetByIDCached(*user.BranchID)
	if err != nil {
		return nil, ErrBranchNotFound
	}

	// === ANTI-FRAUD PRE-CHECKS ===

	// [Feature 1] GPS Accuracy Check — reject fake GPS
	if input.Lat != nil && input.Lng != nil {
		if err := s.antiFraud.ValidateGPSAccuracy(input.AccuracyM); err != nil {
			s.antiFraud.CreateAlert(input.UserID, branch.ID, models.FraudGPSAccuracy, "warning",
				fmt.Sprintf("GPS accuracy: %.1fm", safeFloat(input.AccuracyM)),
				map[string]interface{}{"accuracy": safeFloat(input.AccuracyM)}, input.IP, input.Lat, input.Lng)
			return nil, err
		}
	}

	// [Feature 2] TOTP Single-Use Nonce — prevent QR screenshot relay
	if input.TOTPCode != "" {
		if err := s.antiFraud.CheckTOTPNonce(branch.ID, input.TOTPCode); err != nil {
			s.antiFraud.CreateAlert(input.UserID, branch.ID, models.FraudTOTPReuse, "critical",
				"TOTP code reused", map[string]interface{}{"code": input.TOTPCode}, input.IP, input.Lat, input.Lng)
			return nil, err
		}
	}

	// [Feature 4] Impossible Travel Detection
	if input.Lat != nil && input.Lng != nil {
		now0 := timezone.Now()
		if err := s.antiFraud.CheckImpossibleTravel(input.UserID, *input.Lat, *input.Lng, now0); err != nil {
			s.antiFraud.CreateAlert(input.UserID, branch.ID, models.FraudImpossibleTravel, "critical",
				"Impossible travel detected", map[string]interface{}{"lat": *input.Lat, "lng": *input.Lng}, input.IP, input.Lat, input.Lng)
			return nil, err
		}
	}

	// [Feature 5] Device Fingerprinting — detect buddy punching
	if input.DeviceFingerprint != "" {
		isNew, err := s.antiFraud.CheckDevice(input.UserID, input.DeviceFingerprint, input.UserAgent)
		if err != nil {
			return nil, err // Device blocked
		}
		if isNew {
			s.antiFraud.CreateAlert(input.UserID, branch.ID, models.FraudNewDevice, "warning",
				"New device detected", map[string]interface{}{"ua": input.UserAgent}, input.IP, input.Lat, input.Lng)
		}
	}

	// === STANDARD METHOD VALIDATION ===

	// 2. Validate methods
	result := &LogTimeResult{}
	var validationErrors []string

	if s.branchService.HasMethod(branch, models.MethodQRTOTP) {
		if input.TOTPCode == "" {
			validationErrors = append(validationErrors, "QR/TOTP code required")
		} else if input.ScannedBranchID != "" && input.ScannedBranchID != branch.ID {
			validationErrors = append(validationErrors, "mã QR không thuộc chi nhánh này")
		} else {
			valid, err := s.totpService.ValidateCode(branch.TOTPSecret, input.TOTPCode)
			if err != nil {
				log.Printf("[service][attendance] TOTP validation error for branch %s: %v", branch.ID, err)
				validationErrors = append(validationErrors, "TOTP validation failed")
			} else if !valid {
				validationErrors = append(validationErrors, "mã QR không hợp lệ hoặc đã hết hạn")
			} else {
				result.TOTPVerified = true
			}
		}
	}

	if s.branchService.HasMethod(branch, models.MethodIP) || s.branchService.HasMethod(branch, models.MethodWiFiGPS) {
		if s.ipValidator.Validate(input.IP, branch.IPWhitelist) {
			result.IPVerified = true
		} else {
			validationErrors = append(validationErrors, fmt.Sprintf("IP %s không nằm trong danh sách trắng", input.IP))
		}
	}

	if s.branchService.HasMethod(branch, models.MethodLocation) || s.branchService.HasMethod(branch, models.MethodWiFiGPS) {
		if input.Lat != nil && input.Lng != nil {
			if s.locValidator.Validate(*input.Lat, *input.Lng, branch.Locations) {
				result.LocVerified = true
			} else {
				validationErrors = append(validationErrors, "vị trí nằm ngoài khu vực cho phép")
			}
		} else {
			validationErrors = append(validationErrors, "yêu cầu tọa độ GPS")
		}
	}

	// Face recognition (mock — manager only, always passes if flag is set)
	if s.branchService.HasMethod(branch, models.MethodFace) && input.FaceVerified {
		result.FaceVerified = true
	}

	// Password check
	if s.branchService.HasMethod(branch, models.MethodPassword) && input.PasswordVerified {
		result.PasswordVerified = true
	}

	// Combined WiFi + GPS check (Both must pass)
	if s.branchService.HasMethod(branch, models.MethodWiFiGPS) {
		if result.IPVerified && result.LocVerified {
			result.WiFiGPSVerified = true
		}
	}

	if branch.RequireBiometric && !input.BiometricVerified {
		validationErrors = append(validationErrors, "Yêu cầu xác thực vân tay/FaceID")
	}

	// 3. Final decision
	anyPassed := (result.TOTPVerified && s.branchService.HasMethod(branch, models.MethodQRTOTP)) ||
		(result.IPVerified && s.branchService.HasMethod(branch, models.MethodIP)) ||
		(result.LocVerified && s.branchService.HasMethod(branch, models.MethodLocation)) ||
		(result.FaceVerified && s.branchService.HasMethod(branch, models.MethodFace)) ||
		(result.PasswordVerified && s.branchService.HasMethod(branch, models.MethodPassword)) ||
		(result.WiFiGPSVerified && s.branchService.HasMethod(branch, models.MethodWiFiGPS)) ||
		(result.NFCVerified && s.branchService.HasMethod(branch, models.MethodNFC))

	if !anyPassed {
		log.Printf("[service][attendance] log denied for user %s: branch_id=%s, IP_ok=%v, Loc_ok=%v, WiFiGPS_ok=%v, errors=%v",
			input.UserID, branch.ID, result.IPVerified, result.LocVerified, result.WiFiGPSVerified, validationErrors)
		if len(validationErrors) > 0 {
			return nil, fmt.Errorf("%s", validationErrors[0])
		}
		return nil, ErrMethodRequired
	}

	// [Feature 2] Mark TOTP code as used after successful validation
	if result.TOTPVerified {
		s.antiFraud.MarkTOTPUsed(branch.ID, input.TOTPCode)
	}

	// [Feature 6] IP-Location Cross-Check — detect VPN
	if input.Lat != nil && input.Lng != nil && len(branch.Locations) > 0 {
		branchLoc := branch.Locations[0]
		if err := s.antiFraud.CheckIPLocationConsistency(input.IP, *input.Lat, *input.Lng, branchLoc.Lat, branchLoc.Lng); err != nil {
			s.antiFraud.CreateAlert(input.UserID, branch.ID, models.FraudIPLocationMismatch, "warning",
				"IP-GPS location mismatch", map[string]interface{}{"ip": input.IP, "lat": *input.Lat, "lng": *input.Lng},
				input.IP, input.Lat, input.Lng)
			// Warning only — don't block
			log.Printf("[service][attendance] IP-location mismatch for user %s (warning only)", input.UserID)
		}
	}

	// 4. Resolve shift
	now := timezone.Now()
	workDate := now.Format("2006-01-02")
	method := buildMethodStr(result)

	var shiftID *string
	gracePeriod := 15
	workStartTime := branch.WorkStartTime

	shift, err := s.shiftRepo.FindUserShift(input.UserID, *user.BranchID, workDate)
	if err == nil && shift != nil {
		shiftID = &shift.ID
		gracePeriod = shift.GracePeriodMinutes
		workStartTime = shift.StartTime
	}

	// [Feature 8] Anomaly Detection — flag suspicious check-in times
	anomalyFlag, anomalyScore := s.antiFraud.CheckTimeAnomaly(input.UserID, now)
	if anomalyFlag {
		s.antiFraud.CreateAlert(input.UserID, branch.ID, models.FraudAnomalyTime, "warning",
			fmt.Sprintf("Unusual check-in time (z-score: %.1f)", anomalyScore),
			map[string]interface{}{"time": now.Format("15:04"), "zscore": anomalyScore}, input.IP, input.Lat, input.Lng)
	}
	result.AnomalyFlag = anomalyFlag
	result.AnomalyScore = anomalyScore

	// 5. Create AttendanceLog
	attLog := &models.AttendanceLog{
		UserID:            input.UserID,
		BranchID:          *user.BranchID,
		ShiftID:           shiftID,
		WorkDate:          workDate,
		LoggedAt:          now,
		Method:            method,
		IPAddress:         input.IP,
		Lat:               input.Lat,
		Lng:               input.Lng,
		AccuracyM:         input.AccuracyM,
		DeviceFingerprint: input.DeviceFingerprint,
		TOTPVerified:      result.TOTPVerified,
		IPVerified:        result.IPVerified,
		LocVerified:       result.LocVerified,
		FaceVerified:      result.FaceVerified,
		NFCVerified:       result.NFCVerified,
		PasswordVerified:  input.PasswordVerified,
		BiometricVerified: input.BiometricVerified,
		AnomalyFlag:       anomalyFlag,
		AnomalyScore:      anomalyScore,
	}
	if err := s.logRepo.Create(attLog); err != nil {
		return nil, fmt.Errorf("create attendance log: %w", err)
	}
	result.Log = attLog

	// 5. Find or create daily Attendance summary
	att, err := s.attendanceRepo.FindTodayByUser(input.UserID)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		// First log of the day → create attendance
		att = &models.Attendance{
			UserID:       input.UserID,
			BranchID:     *user.BranchID,
			ShiftID:      shiftID,
			WorkDate:     workDate,
			CheckInAt:    &now,
			Status:       calculateStatusWithGrace(now, workStartTime, gracePeriod),
			Method:       method,
			IPAddress:    input.IP,
			Lat:          input.Lat,
			Lng:          input.Lng,
			TOTPVerified:     result.TOTPVerified,
			IPVerified:       result.IPVerified,
			LocVerified:      result.LocVerified,
			FaceVerified:     result.FaceVerified,
			NFCVerified:      result.NFCVerified,
			PasswordVerified: result.PasswordVerified,
		}
		if err := s.attendanceRepo.Create(att); err != nil {
			return nil, fmt.Errorf("create attendance: %w", err)
		}
	} else if err == nil {
		// Subsequent log → update attendance summary
		// CheckInAt = earliest, CheckOutAt = latest
		if att.CheckInAt != nil && now.Before(*att.CheckInAt) {
			att.CheckInAt = &now
			att.Status = calculateStatusWithGrace(now, workStartTime, gracePeriod)
		}
		if att.CheckOutAt == nil || now.After(*att.CheckOutAt) {
			att.CheckOutAt = &now
		}
		// If only 1 previous log and this is the 2nd, set CheckOutAt
		if att.CheckOutAt == nil {
			att.CheckOutAt = &now
		}
		if err := s.attendanceRepo.Update(att); err != nil {
			return nil, fmt.Errorf("update attendance: %w", err)
		}
	} else {
		return nil, fmt.Errorf("find attendance: %w", err)
	}


	// 7. Count today's logs
	count, _ := s.logRepo.CountTodayLogs(input.UserID, workDate)
	result.Attendance = att
	result.LogCount = int(count)

	log.Printf("[service][attendance] log-time: user=%s branch=%s method=%s log#%d", input.UserID, branch.Name, method, count)
	return result, nil
}

// GetTodayStatus returns today's attendance summary for a user.
func (s *AttendanceService) GetTodayStatus(userID string) (*models.Attendance, error) {
	att, err := s.attendanceRepo.FindTodayByUser(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return att, nil
}

// GetRecentByBranch returns the most recent attendance records for a branch.
func (s *AttendanceService) GetRecentByBranch(branchID string, limit int) ([]repository.RecentCheckIn, error) {
	return s.attendanceRepo.RecentCheckIns(branchID, limit)
}

// GetTodayLogs returns all time logs for a user today.
func (s *AttendanceService) GetTodayLogs(userID string) ([]models.AttendanceLog, error) {
	workDate := timezone.Now().Format("2006-01-02")
	return s.logRepo.FindTodayLogs(userID, workDate)
}

func (s *AttendanceService) List(params repository.AttendanceListParams) (*repository.AttendanceListResult, error) {
	return s.attendanceRepo.List(params)
}

// --- Helpers ---

func calculateStatusWithGrace(checkInTime time.Time, workStartTime string, gracePeriodMinutes int) models.AttendanceStatus {
	if workStartTime == "" {
		workStartTime = "08:00"
	}
	if gracePeriodMinutes < 0 {
		gracePeriodMinutes = 15
	}

	var hour, min int
	fmt.Sscanf(workStartTime, "%d:%d", &hour, &min)

	deadline := time.Date(checkInTime.Year(), checkInTime.Month(), checkInTime.Day(), hour, min, 0, 0, checkInTime.Location())
	deadline = deadline.Add(time.Duration(gracePeriodMinutes) * time.Minute)

	if checkInTime.Before(deadline) {
		return models.StatusOnTime
	}
	return models.StatusLate
}

func buildMethodStr(result *LogTimeResult) string {
	s := ""
	if result.TOTPVerified {
		s += "qr_totp"
	}
	if result.IPVerified {
		if s != "" {
			s += ","
		}
		s += "ip"
	}
	if result.LocVerified {
		if s != "" {
			s += ","
		}
		s += "location"
	}
	if result.FaceVerified {
		if s != "" {
			s += ","
		}
		s += "face"
	}
	if result.NFCVerified {
		if s != "" {
			s += ","
		}
		s += "nfc"
	}
	if result.PasswordVerified {
		if s != "" {
			s += ","
		}
		s += "password"
	}
	return s
}

func safeFloat(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}
