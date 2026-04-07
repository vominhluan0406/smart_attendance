package service

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/repository"
	"github.com/smart-attendance/smart-attendance/internal/timezone"
)

var (
	ErrAdjustmentNotFound     = errors.New("không tìm thấy yêu cầu bổ sung công")
	ErrAdjustmentDuplicate    = errors.New("đã có yêu cầu bổ sung công đang chờ duyệt cho ngày này")
	ErrAdjustmentFutureDate   = errors.New("không thể bổ sung công cho ngày trong tương lai")
	ErrAdjustmentInvalidTime  = errors.New("giờ check-out phải sau giờ check-in")
	ErrAdjustmentMissingTime  = errors.New("vui lòng nhập ít nhất giờ check-in hoặc check-out")
)

type AttendanceAdjustmentService struct {
	adjRepo        *repository.AttendanceAdjustmentRepository
	attendanceRepo *repository.AttendanceRepository
	userRepo       *repository.UserRepository
}

func NewAttendanceAdjustmentService(
	adjRepo *repository.AttendanceAdjustmentRepository,
	attendanceRepo *repository.AttendanceRepository,
	userRepo *repository.UserRepository,
) *AttendanceAdjustmentService {
	return &AttendanceAdjustmentService{
		adjRepo:        adjRepo,
		attendanceRepo: attendanceRepo,
		userRepo:       userRepo,
	}
}

// CreateRequest creates a new attendance adjustment request from an employee.
func (s *AttendanceAdjustmentService) CreateRequest(userID, workDate, checkInStr, checkOutStr, reason string) error {
	// Validate work date
	wd, err := time.Parse("2006-01-02", workDate)
	if err != nil {
		return fmt.Errorf("ngày không hợp lệ")
	}
	today := timezone.Now().Truncate(24 * time.Hour)
	if wd.After(today) {
		return ErrAdjustmentFutureDate
	}

	// Parse check-in/check-out times
	var checkIn, checkOut *time.Time
	if checkInStr != "" {
		t, err := time.Parse("15:04", checkInStr)
		if err != nil {
			return fmt.Errorf("giờ check-in không hợp lệ")
		}
		ci := time.Date(wd.Year(), wd.Month(), wd.Day(), t.Hour(), t.Minute(), 0, 0, timezone.VN)
		checkIn = &ci
	}
	if checkOutStr != "" {
		t, err := time.Parse("15:04", checkOutStr)
		if err != nil {
			return fmt.Errorf("giờ check-out không hợp lệ")
		}
		co := time.Date(wd.Year(), wd.Month(), wd.Day(), t.Hour(), t.Minute(), 0, 0, timezone.VN)
		checkOut = &co
	}

	if checkIn == nil && checkOut == nil {
		return ErrAdjustmentMissingTime
	}
	if checkIn != nil && checkOut != nil && !checkOut.After(*checkIn) {
		return ErrAdjustmentInvalidTime
	}

	if reason == "" {
		return fmt.Errorf("vui lòng nhập lý do")
	}

	// Check duplicate pending request
	_, err = s.adjRepo.FindPendingByUserAndDate(userID, workDate)
	if err == nil {
		return ErrAdjustmentDuplicate
	}

	// Find existing attendance record (may be nil)
	var attendanceID *string
	existing, err := s.attendanceRepo.FindTodayByUserAndDate(userID, workDate)
	if err == nil && existing != nil {
		attendanceID = &existing.ID
	}

	adj := &models.AttendanceAdjustment{
		UserID:            userID,
		AttendanceID:      attendanceID,
		WorkDate:          workDate,
		RequestedCheckIn:  checkIn,
		RequestedCheckOut: checkOut,
		Reason:            reason,
		Status:            models.AdjustStatusPending,
	}

	if err := s.adjRepo.Create(adj); err != nil {
		log.Printf("[service][adjustment] ERROR create: user=%s date=%s err=%v", userID, workDate, err)
		return fmt.Errorf("không thể tạo yêu cầu")
	}

	log.Printf("[service][adjustment] request created: user=%s date=%s", userID, workDate)
	return nil
}

// GetMyRequests returns the employee's own adjustment requests.
func (s *AttendanceAdjustmentService) GetMyRequests(userID string, page, limit int) (*repository.AdjustmentListResult, error) {
	return s.adjRepo.List(repository.AdjustmentListParams{
		Page:   page,
		Limit:  limit,
		UserID: userID,
	})
}

// GetBranchRequests returns adjustment requests for a branch (manager view).
func (s *AttendanceAdjustmentService) GetBranchRequests(branchID, status string, page, limit int) (*repository.AdjustmentListResult, error) {
	return s.adjRepo.List(repository.AdjustmentListParams{
		Page:     page,
		Limit:    limit,
		BranchID: branchID,
		Status:   status,
	})
}

// ReviewRequest approves or rejects an adjustment request.
// On approval, updates the actual attendance record.
func (s *AttendanceAdjustmentService) ReviewRequest(reviewerID, requestID string, status models.AdjustmentStatus, note string) error {
	adj, err := s.adjRepo.FindByID(requestID)
	if err != nil {
		return ErrAdjustmentNotFound
	}

	if adj.Status != models.AdjustStatusPending {
		return fmt.Errorf("yêu cầu này đã được xử lý")
	}

	now := timezone.Now()
	adj.Status = status
	adj.ReviewerID = &reviewerID
	adj.ReviewedAt = &now
	adj.ReviewerNote = note

	if err := s.adjRepo.Update(adj); err != nil {
		log.Printf("[service][adjustment] ERROR review update: id=%s err=%v", requestID, err)
		return fmt.Errorf("không thể cập nhật yêu cầu")
	}

	if status == models.AdjustStatusApproved {
		s.applyAdjustment(adj)
	}

	log.Printf("[service][adjustment] reviewed: id=%s status=%s by=%s", requestID, status, reviewerID)
	return nil
}

// applyAdjustment updates the actual attendance record with the requested times.
func (s *AttendanceAdjustmentService) applyAdjustment(adj *models.AttendanceAdjustment) {
	existing, err := s.attendanceRepo.FindTodayByUserAndDate(adj.UserID, adj.WorkDate)
	if err != nil || existing == nil {
		// No attendance record exists — create one
		user, err := s.userRepo.FindByID(adj.UserID)
		if err != nil || user.BranchID == nil {
			log.Printf("[service][adjustment] ERROR applyAdjustment: user %s has no branch_id", adj.UserID)
			return
		}

		att := &models.Attendance{
			UserID:       adj.UserID,
			BranchID:     *user.BranchID,
			WorkDate:     adj.WorkDate,
			CheckInAt:    adj.RequestedCheckIn,
			CheckOutAt:   adj.RequestedCheckOut,
			Status:       models.StatusOnTime,
			Method:       "adjustment",
			IsAdjusted:   true,
			AdjustedByID: adj.ReviewerID,
			Note:         fmt.Sprintf("Bổ sung công: %s", adj.Reason),
		}
		if err := s.attendanceRepo.Create(att); err != nil {
			log.Printf("[service][adjustment] ERROR create attendance: %v", err)
		} else {
			log.Printf("[service][adjustment] created attendance for %s on %s", adj.UserID, adj.WorkDate)
		}
		return
	}

	// Update existing record
	if adj.RequestedCheckIn != nil {
		existing.CheckInAt = adj.RequestedCheckIn
	}
	if adj.RequestedCheckOut != nil {
		existing.CheckOutAt = adj.RequestedCheckOut
	}
	existing.IsAdjusted = true
	existing.AdjustedByID = adj.ReviewerID
	if existing.Note != "" {
		existing.Note += " | "
	}
	existing.Note += fmt.Sprintf("Bổ sung công: %s", adj.Reason)

	// Re-evaluate status if was absent/invalidated
	if existing.Status == models.StatusAbsent || existing.Status == models.StatusInvalidated {
		existing.Status = models.StatusOnTime
	}

	if err := s.attendanceRepo.Update(existing); err != nil {
		log.Printf("[service][adjustment] ERROR update attendance: %v", err)
	} else {
		log.Printf("[service][adjustment] updated attendance for %s on %s", adj.UserID, adj.WorkDate)
	}
}
