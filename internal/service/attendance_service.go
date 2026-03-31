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
	}
}

// LogTimeInput is the input for each QR scan / time log.
type LogTimeInput struct {
	UserID   string   `json:"user_id"`
	TOTPCode string   `json:"totp_code"`
	IP       string   `json:"ip"`
	Lat      *float64 `json:"lat"`
	Lng      *float64 `json:"lng"`
}

// LogTimeResult is returned after a successful time log.
type LogTimeResult struct {
	Log          *models.AttendanceLog `json:"log"`
	Attendance   *models.Attendance    `json:"attendance"`
	TOTPVerified bool                  `json:"totp_verified"`
	IPVerified   bool                  `json:"ip_verified"`
	LocVerified  bool                  `json:"loc_verified"`
	LogCount     int                   `json:"log_count"` // total logs today
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

	// 2. Validate methods (same as before)
	result := &LogTimeResult{}
	var validationErrors []string

	if s.branchService.HasMethod(branch, models.MethodQRTOTP) {
		if input.TOTPCode == "" {
			validationErrors = append(validationErrors, "QR/TOTP code required")
		} else {
			valid, err := s.totpService.ValidateCode(branch.TOTPSecret, input.TOTPCode)
			if err != nil {
				log.Printf("[service][attendance] TOTP validation error for branch %s: %v", branch.ID, err)
				validationErrors = append(validationErrors, "TOTP validation failed")
			} else if !valid {
				validationErrors = append(validationErrors, "invalid or expired QR code")
			} else {
				result.TOTPVerified = true
			}
		}
	}

	if s.branchService.HasMethod(branch, models.MethodIP) {
		if s.ipValidator.Validate(input.IP, branch.IPWhitelist) {
			result.IPVerified = true
		} else {
			validationErrors = append(validationErrors, fmt.Sprintf("IP %s not in whitelist", input.IP))
		}
	}

	if s.branchService.HasMethod(branch, models.MethodLocation) {
		if input.Lat != nil && input.Lng != nil {
			if s.locValidator.Validate(*input.Lat, *input.Lng, branch.Locations) {
				result.LocVerified = true
			} else {
				validationErrors = append(validationErrors, "location outside allowed area")
			}
		} else {
			validationErrors = append(validationErrors, "GPS location required")
		}
	}

	if !result.TOTPVerified && !result.IPVerified && !result.LocVerified {
		log.Printf("[service][attendance] log denied for user %s: %v", input.UserID, validationErrors)
		if len(validationErrors) > 0 {
			return nil, fmt.Errorf("%s", validationErrors[0])
		}
		return nil, ErrMethodRequired
	}

	// 3. Resolve shift
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

	// 4. Create AttendanceLog
	attLog := &models.AttendanceLog{
		UserID:       input.UserID,
		BranchID:     *user.BranchID,
		ShiftID:      shiftID,
		WorkDate:     workDate,
		LoggedAt:     now,
		Method:       method,
		IPAddress:    input.IP,
		Lat:          input.Lat,
		Lng:          input.Lng,
		TOTPVerified: result.TOTPVerified,
		IPVerified:   result.IPVerified,
		LocVerified:  result.LocVerified,
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
			TOTPVerified: result.TOTPVerified,
			IPVerified:   result.IPVerified,
			LocVerified:  result.LocVerified,
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

	// 6. Count today's logs
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
	return s
}
