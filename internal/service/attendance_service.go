package service

import (
	"errors"
	"fmt"
	"log"
	"strings"
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
	user, branch, err := s.validateUserAndBranch(input.UserID)
	if err != nil {
		return nil, err
	}

	// 2. Anti-fraud pre-checks
	if err := s.performAntiFraudPreChecks(user, branch, input); err != nil {
		return nil, err
	}

	// 3. Standard method validation
	result, validationErrors, err := s.validateVerificationMethods(branch, input)
	if err != nil {
		return nil, err
	}

	// Check if at least one method passed
	if !s.isAnyMethodPassed(branch, result, input) {
		log.Printf("[service][attendance] log denied for user %s: branch_id=%s, errors=%v",
			input.UserID, branch.ID, validationErrors)
		if len(validationErrors) > 0 {
			return nil, fmt.Errorf("%s", validationErrors[0])
		}
		return nil, ErrMethodRequired
	}

	// 4. Mark TOTP as used if successful
	if result.TOTPVerified {
		s.antiFraud.MarkTOTPUsed(branch.ID, input.TOTPCode)
	}

	// 5. Anti-fraud post-checks (VPN detection)
	s.performAntiFraudPostChecks(user, branch, input)

	// 6. Resolve shift and anomaly detection
	now := timezone.Now()
	shiftData, err := s.resolveShiftData(input.UserID, branch, now)
	if err != nil {
		return nil, err
	}
	result.AnomalyFlag = shiftData.anomalyFlag
	result.AnomalyScore = shiftData.anomalyScore

	// 7. Create AttendanceLog
	method := buildMethodStr(result)
	attLog := s.buildAttendanceLog(input, branch, result, shiftData, method, now)
	if err := s.logRepo.Create(attLog); err != nil {
		return nil, fmt.Errorf("failed to create attendance log: %w", err)
	}
	result.Log = attLog

	// 8. Update daily Attendance summary
	att, err := s.updateDailyAttendance(input.UserID, branch, shiftData, method, now, input)
	if err != nil {
		return nil, err
	}

	// 9. Count today's logs
	count, _ := s.logRepo.CountTodayLogs(input.UserID, shiftData.workDate)
	result.Attendance = att
	result.LogCount = int(count)

	log.Printf("[service][attendance] log-time: user=%s branch=%s method=%s log#%d", input.UserID, branch.Name, method, count)
	return result, nil
}

// --- Decomposition Helpers ---

func (s *AttendanceService) validateUserAndBranch(userID string) (*models.User, *models.Branch, error) {
	user, err := s.userService.GetByID(userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to identify user: %w", ErrUserNotFound)
	}
	if user.BranchID == nil {
		return nil, nil, fmt.Errorf("user not assigned: %w", ErrNoBranch)
	}

	branch, err := s.branchService.GetByIDCached(*user.BranchID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load branch: %w", ErrBranchNotFound)
	}
	return user, branch, nil
}

func (s *AttendanceService) performAntiFraudPreChecks(user *models.User, branch *models.Branch, input LogTimeInput) error {
	// GPS Accuracy Check
	if input.Lat != nil && input.Lng != nil {
		if err := s.antiFraud.ValidateGPSAccuracy(input.AccuracyM); err != nil {
			s.antiFraud.CreateAlert(user.ID, branch.ID, models.FraudGPSAccuracy, "warning",
				fmt.Sprintf("GPS accuracy: %.1fm", safeFloat(input.AccuracyM)),
				map[string]interface{}{"accuracy": safeFloat(input.AccuracyM)}, input.IP, input.Lat, input.Lng)
			return fmt.Errorf("gps accuracy validation failed: %w", err)
		}
	}

	// TOTP Single-Use Nonce
	if input.TOTPCode != "" {
		if err := s.antiFraud.CheckTOTPNonce(branch.ID, input.TOTPCode); err != nil {
			s.antiFraud.CreateAlert(user.ID, branch.ID, models.FraudTOTPReuse, "critical",
				"TOTP code reused", map[string]interface{}{"code": input.TOTPCode}, input.IP, input.Lat, input.Lng)
			return fmt.Errorf("totp reuse detection: %w", err)
		}
	}

	// Impossible Travel Detection
	if input.Lat != nil && input.Lng != nil {
		if err := s.antiFraud.CheckImpossibleTravel(user.ID, *input.Lat, *input.Lng, timezone.Now()); err != nil {
			s.antiFraud.CreateAlert(user.ID, branch.ID, models.FraudImpossibleTravel, "critical",
				"Impossible travel detected", map[string]interface{}{"lat": *input.Lat, "lng": *input.Lng}, input.IP, input.Lat, input.Lng)
			return fmt.Errorf("travel detection: %w", err)
		}
	}

	// Device Fingerprinting
	if input.DeviceFingerprint != "" {
		isNew, err := s.antiFraud.CheckDevice(user.ID, input.DeviceFingerprint, input.UserAgent)
		if err != nil {
			return fmt.Errorf("device access denied: %w", err)
		}
		if isNew {
			s.antiFraud.CreateAlert(user.ID, branch.ID, models.FraudNewDevice, "warning",
				"New device detected", map[string]interface{}{"ua": input.UserAgent}, input.IP, input.Lat, input.Lng)
		}
	}
	return nil
}

func (s *AttendanceService) validateVerificationMethods(branch *models.Branch, input LogTimeInput) (*LogTimeResult, []string, error) {
	result := &LogTimeResult{}
	var validationErrors []string

	// QR/TOTP
	if s.branchService.HasMethod(branch, models.MethodQRTOTP) {
		if input.TOTPCode == "" {
			validationErrors = append(validationErrors, "QR/TOTP code required")
		} else if input.ScannedBranchID != "" && input.ScannedBranchID != branch.ID {
			validationErrors = append(validationErrors, "mã QR không thuộc chi nhánh này")
		} else {
			valid, err := s.totpService.ValidateCode(branch.TOTPSecret, input.TOTPCode)
			if err != nil {
				return nil, nil, fmt.Errorf("totp validation system error: %w", err)
			} else if !valid {
				validationErrors = append(validationErrors, "mã QR không hợp lệ hoặc đã hết hạn")
			} else {
				result.TOTPVerified = true
			}
		}
	}

	// IP
	if s.branchService.HasMethod(branch, models.MethodIP) || s.branchService.HasMethod(branch, models.MethodWiFiGPS) {
		if s.ipValidator.Validate(input.IP, branch.IPWhitelist) {
			result.IPVerified = true
		} else {
			validationErrors = append(validationErrors, fmt.Sprintf("IP %s không nằm trong danh sách trắng", input.IP))
		}
	}

	// GPS Location
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

	// Face, Password, NFC, Biometric
	if s.branchService.HasMethod(branch, models.MethodFace) && input.FaceVerified {
		result.FaceVerified = true
	}
	if s.branchService.HasMethod(branch, models.MethodPassword) && input.PasswordVerified {
		result.PasswordVerified = true
	}
	if s.branchService.HasMethod(branch, models.MethodNFC) && input.NFCVerified {
		result.NFCVerified = true
	}
	if s.branchService.HasMethod(branch, models.MethodWiFiGPS) && result.IPVerified && result.LocVerified {
		result.WiFiGPSVerified = true
	}
	if branch.RequireBiometric && !input.BiometricVerified {
		validationErrors = append(validationErrors, "Yêu cầu xác thực vân tay/FaceID")
	}

	return result, validationErrors, nil
}

func (s *AttendanceService) isAnyMethodPassed(branch *models.Branch, result *LogTimeResult, input LogTimeInput) bool {
	// Password check-in is performed on a supervised manager device.
	// If successful, it bypasses biometric and other constraints.
	if result.PasswordVerified {
		return true
	}

	// Check biometric constraint first if required
	if branch.RequireBiometric && !input.BiometricVerified {
		return false
	}
	return (result.TOTPVerified && s.branchService.HasMethod(branch, models.MethodQRTOTP)) ||
		(result.IPVerified && s.branchService.HasMethod(branch, models.MethodIP)) ||
		(result.LocVerified && s.branchService.HasMethod(branch, models.MethodLocation)) ||
		(result.FaceVerified && s.branchService.HasMethod(branch, models.MethodFace)) ||
		(result.WiFiGPSVerified && s.branchService.HasMethod(branch, models.MethodWiFiGPS)) ||
		(result.NFCVerified && s.branchService.HasMethod(branch, models.MethodNFC))
}

func (s *AttendanceService) performAntiFraudPostChecks(user *models.User, branch *models.Branch, input LogTimeInput) {
	if input.Lat != nil && input.Lng != nil && len(branch.Locations) > 0 {
		branchLoc := branch.Locations[0]
		if err := s.antiFraud.CheckIPLocationConsistency(input.IP, *input.Lat, *input.Lng, branchLoc.Lat, branchLoc.Lng); err != nil {
			s.antiFraud.CreateAlert(user.ID, branch.ID, models.FraudIPLocationMismatch, "warning",
				"IP-GPS location mismatch", map[string]interface{}{"ip": input.IP, "lat": *input.Lat, "lng": *input.Lng},
				input.IP, input.Lat, input.Lng)
		}
	}
}

type shiftData struct {
	shiftID            *string
	gracePeriod        int
	workStartTime      string
	workDate           string
	anomalyFlag       bool
	anomalyScore      float64
}

func (s *AttendanceService) resolveShiftData(userID string, branch *models.Branch, now time.Time) (shiftData, error) {
	data := shiftData{
		gracePeriod:   15,
		workStartTime: branch.WorkStartTime,
		workDate:      now.Format("2006-01-02"),
	}

	shift, err := s.shiftRepo.FindUserShift(userID, branch.ID, data.workDate)
	if err == nil && shift != nil {
		data.shiftID = &shift.ID
		data.gracePeriod = shift.GracePeriodMinutes
		data.workStartTime = shift.StartTime
	}

	data.anomalyFlag, data.anomalyScore = s.antiFraud.CheckTimeAnomaly(userID, now)
	return data, nil
}

func (s *AttendanceService) buildAttendanceLog(input LogTimeInput, branch *models.Branch, result *LogTimeResult, data shiftData, method string, now time.Time) *models.AttendanceLog {
	return &models.AttendanceLog{
		UserID:            input.UserID,
		BranchID:          branch.ID,
		ShiftID:           data.shiftID,
		WorkDate:          data.workDate,
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
		AnomalyFlag:       data.anomalyFlag,
		AnomalyScore:      data.anomalyScore,
	}
}

func (s *AttendanceService) updateDailyAttendance(userID string, branch *models.Branch, data shiftData, method string, now time.Time, input LogTimeInput) (*models.Attendance, error) {
	att, err := s.attendanceRepo.FindTodayByUser(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			att = &models.Attendance{
				UserID:       userID,
				BranchID:     branch.ID,
				ShiftID:      data.shiftID,
				WorkDate:     data.workDate,
				CheckInAt:    &now,
				Status:       calculateStatusWithGrace(now, data.workStartTime, data.gracePeriod),
				Method:       method,
				IPAddress:    input.IP,
				Lat:          input.Lat,
				Lng:          input.Lng,
				TOTPVerified:     resultVerified(method, "qr_totp"),
				IPVerified:       resultVerified(method, "ip"),
				LocVerified:      resultVerified(method, "location"),
				FaceVerified:     resultVerified(method, "face"),
				NFCVerified:      resultVerified(method, "nfc"),
				PasswordVerified: resultVerified(method, "password"),
			}
			if err := s.attendanceRepo.Create(att); err != nil {
				return nil, fmt.Errorf("failed to initialize daily attendance: %w", err)
			}
			return att, nil
		}
		return nil, fmt.Errorf("database error during attendance lookup: %w", err)
	}

	// Update existing record (Immutability note: In Go GORM context, we usually update the fetched pointer)
	if att.CheckInAt != nil && now.Before(*att.CheckInAt) {
		att.CheckInAt = &now
		att.Status = calculateStatusWithGrace(now, data.workStartTime, data.gracePeriod)
	}
	if att.CheckOutAt == nil || now.After(*att.CheckOutAt) {
		att.CheckOutAt = &now
	}
	if err := s.attendanceRepo.Update(att); err != nil {
		return nil, fmt.Errorf("failed to update daily attendance: %w", err)
	}
	return att, nil
}

func resultVerified(methodStr, search string) bool {
	return strings.Contains(methodStr, search)
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

func (s *AttendanceService) InvalidateLog(logID string, reviewerID *string, note string) error {
	logObj, err := s.logRepo.FindByID(logID)
	if err != nil {
		return fmt.Errorf("không tìm thấy log")
	}
	if err := s.logRepo.Invalidate(logID, reviewerID, note); err != nil {
		return err
	}
	return s.RecalculateDailyAttendance(logObj.UserID, logObj.WorkDate)
}

func (s *AttendanceService) RecalculateDailyAttendance(userID string, workDate string) error {
	logs, err := s.logRepo.FindTodayLogs(userID, workDate)
	if err != nil {
		return err
	}

	var validLogs []models.AttendanceLog
	for _, l := range logs {
		if !l.IsInvalidated {
			validLogs = append(validLogs, l)
		}
	}

	att, err := s.attendanceRepo.FindTodayByUserAndDate(userID, workDate)
	if err != nil {
		return err
	}

	if len(validLogs) == 0 {
		att.Status = models.StatusInvalidated
		att.CheckInAt = nil
		att.CheckOutAt = nil
	} else {
		first := validLogs[0]
		last := validLogs[len(validLogs)-1]

		att.CheckInAt = &first.LoggedAt
		att.CheckOutAt = &last.LoggedAt

		// Re-evaluate status if it was invalidated
		if att.Status == models.StatusInvalidated {
			user, err := s.userService.GetByID(userID)
			if err == nil && user.BranchID != nil {
				branch, err := s.branchService.GetByIDCached(*user.BranchID)
				if err == nil {
					shiftData, _ := s.resolveShiftData(userID, branch, *att.CheckInAt)
					att.Status = calculateStatusWithGrace(*att.CheckInAt, shiftData.workStartTime, shiftData.gracePeriod)
				}
			}
		}
	}

	return s.attendanceRepo.Update(att)
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
	var methods []string
	if result.TOTPVerified {
		methods = append(methods, "qr_totp")
	}
	if result.IPVerified {
		methods = append(methods, "ip")
	}
	if result.LocVerified {
		methods = append(methods, "location")
	}
	if result.FaceVerified {
		methods = append(methods, "face")
	}
	if result.NFCVerified {
		methods = append(methods, "nfc")
	}
	if result.PasswordVerified {
		methods = append(methods, "password")
	}
	return strings.Join(methods, ",")
}

func safeFloat(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}
