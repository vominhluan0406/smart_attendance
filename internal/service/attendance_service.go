package service

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrAlreadyCheckedIn  = errors.New("already checked in today")
	ErrNotCheckedIn      = errors.New("not checked in today")
	ErrAlreadyCheckedOut = errors.New("already checked out")
	ErrTOTPInvalid       = errors.New("invalid QR/TOTP code")
	ErrIPNotAllowed      = errors.New("IP address not in whitelist")
	ErrLocationOutside   = errors.New("location outside allowed area")
	ErrNoBranch          = errors.New("user not assigned to any branch")
	ErrMethodRequired    = errors.New("at least one verification method must pass")
)

type AttendanceService struct {
	attendanceRepo *repository.AttendanceRepository
	branchService  *BranchService
	userService    *UserService
	totpService    *TOTPService
	ipValidator    *IPValidator
	locValidator   *LocationValidator
}

func NewAttendanceService(
	attendanceRepo *repository.AttendanceRepository,
	branchService *BranchService,
	userService *UserService,
	totpService *TOTPService,
	ipValidator *IPValidator,
	locValidator *LocationValidator,
) *AttendanceService {
	return &AttendanceService{
		attendanceRepo: attendanceRepo,
		branchService:  branchService,
		userService:    userService,
		totpService:    totpService,
		ipValidator:    ipValidator,
		locValidator:   locValidator,
	}
}

type CheckInInput struct {
	UserID   string   `json:"user_id"`
	TOTPCode string   `json:"totp_code"`
	IP       string   `json:"ip"`
	Lat      *float64 `json:"lat"`
	Lng      *float64 `json:"lng"`
}

type CheckInResult struct {
	Attendance   *models.Attendance `json:"attendance"`
	TOTPVerified bool               `json:"totp_verified"`
	IPVerified   bool               `json:"ip_verified"`
	LocVerified  bool               `json:"loc_verified"`
}

func (s *AttendanceService) CheckIn(input CheckInInput) (*CheckInResult, error) {
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

	// 2. Check if already checked in today
	existing, err := s.attendanceRepo.FindTodayByUser(input.UserID)
	if err == nil && existing != nil {
		if existing.CheckOutAt != nil {
			return nil, ErrAlreadyCheckedOut
		}
		return nil, ErrAlreadyCheckedIn
	}

	// 3. Validate each method the branch requires
	result := &CheckInResult{}
	var validationErrors []string

	// QR/TOTP validation
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

	// IP validation
	if s.branchService.HasMethod(branch, models.MethodIP) {
		if s.ipValidator.Validate(input.IP, branch.IPWhitelist) {
			result.IPVerified = true
		} else {
			validationErrors = append(validationErrors, fmt.Sprintf("IP %s not in whitelist", input.IP))
		}
	}

	// Location validation
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

	// At least one method must pass
	if !result.TOTPVerified && !result.IPVerified && !result.LocVerified {
		log.Printf("[service][attendance] check-in denied for user %s: %v", input.UserID, validationErrors)
		if len(validationErrors) > 0 {
			return nil, fmt.Errorf("%s", validationErrors[0])
		}
		return nil, ErrMethodRequired
	}

	// 4. Create attendance record
	now := time.Now()
	method := buildMethodString(result)

	att := &models.Attendance{
		UserID:       input.UserID,
		BranchID:     *user.BranchID,
		CheckInAt:    &now,
		Status:       calculateStatus(now, branch.WorkStartTime),
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

	result.Attendance = att
	log.Printf("[service][attendance] check-in: user=%s branch=%s method=%s status=%s", input.UserID, branch.Name, method, att.Status)
	return result, nil
}

func (s *AttendanceService) CheckOut(userID string) (*models.Attendance, error) {
	att, err := s.attendanceRepo.FindTodayByUser(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotCheckedIn
		}
		return nil, err
	}

	if att.CheckOutAt != nil {
		return nil, ErrAlreadyCheckedOut
	}

	now := time.Now()
	att.CheckOutAt = &now

	if err := s.attendanceRepo.Update(att); err != nil {
		return nil, fmt.Errorf("update attendance: %w", err)
	}

	log.Printf("[service][attendance] check-out: user=%s at=%s", userID, now.Format("15:04:05"))
	return att, nil
}

func (s *AttendanceService) GetTodayStatus(userID string) (*models.Attendance, error) {
	att, err := s.attendanceRepo.FindTodayByUser(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Not checked in yet
		}
		return nil, err
	}
	return att, nil
}

func (s *AttendanceService) List(params repository.AttendanceListParams) (*repository.AttendanceListResult, error) {
	return s.attendanceRepo.List(params)
}

// --- Helpers ---

func calculateStatus(checkInTime time.Time, workStartTime string) models.AttendanceStatus {
	if workStartTime == "" {
		workStartTime = "08:00"
	}

	var hour, min int
	fmt.Sscanf(workStartTime, "%d:%d", &hour, &min)

	deadline := time.Date(checkInTime.Year(), checkInTime.Month(), checkInTime.Day(), hour, min, 0, 0, checkInTime.Location())
	// 15 minute grace period
	deadline = deadline.Add(15 * time.Minute)

	if checkInTime.Before(deadline) {
		return models.StatusOnTime
	}
	return models.StatusLate
}

func buildMethodString(result *CheckInResult) string {
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
	s := ""
	for i, m := range methods {
		if i > 0 {
			s += ","
		}
		s += m
	}
	return s
}
