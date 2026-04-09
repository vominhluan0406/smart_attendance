package service

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/smart-attendance/attendance-service/internal/client"
	"github.com/smart-attendance/attendance-service/internal/model"
	"github.com/smart-attendance/attendance-service/internal/repository"
)

var (
	ErrAdjustmentNotFound    = errors.New("adjustment request not found")
	ErrAdjustmentDuplicate   = errors.New("a pending adjustment request already exists for this date")
	ErrAdjustmentFutureDate  = errors.New("cannot adjust attendance for a future date")
	ErrAdjustmentInvalidTime = errors.New("check-out time must be after check-in time")
	ErrAdjustmentMissingTime = errors.New("please provide at least check-in or check-out time")
)

type AttendanceAdjustmentService struct {
	adjRepo        *repository.AttendanceAdjustmentRepository
	attendanceRepo *repository.AttendanceRepository
	authClient     *client.AuthClient
}

func NewAttendanceAdjustmentService(
	adjRepo *repository.AttendanceAdjustmentRepository,
	attendanceRepo *repository.AttendanceRepository,
	authClient *client.AuthClient,
) *AttendanceAdjustmentService {
	return &AttendanceAdjustmentService{
		adjRepo:        adjRepo,
		attendanceRepo: attendanceRepo,
		authClient:     authClient,
	}
}

// CreateRequest creates a new attendance adjustment request from an employee.
func (s *AttendanceAdjustmentService) CreateRequest(userID, workDate, checkInStr, checkOutStr, reason string) error {
	// Validate work date
	wd, err := time.Parse("2006-01-02", workDate)
	if err != nil {
		return fmt.Errorf("invalid date format")
	}
	today := time.Now().Truncate(24 * time.Hour)
	if wd.After(today) {
		return ErrAdjustmentFutureDate
	}

	// Parse check-in/check-out times
	var checkIn, checkOut *time.Time
	loc := time.Now().Location()
	if checkInStr != "" {
		t, err := time.Parse("15:04", checkInStr)
		if err != nil {
			return fmt.Errorf("invalid check-in time format")
		}
		ci := time.Date(wd.Year(), wd.Month(), wd.Day(), t.Hour(), t.Minute(), 0, 0, loc)
		checkIn = &ci
	}
	if checkOutStr != "" {
		t, err := time.Parse("15:04", checkOutStr)
		if err != nil {
			return fmt.Errorf("invalid check-out time format")
		}
		co := time.Date(wd.Year(), wd.Month(), wd.Day(), t.Hour(), t.Minute(), 0, 0, loc)
		checkOut = &co
	}

	if checkIn == nil && checkOut == nil {
		return ErrAdjustmentMissingTime
	}
	if checkIn != nil && checkOut != nil && !checkOut.After(*checkIn) {
		return ErrAdjustmentInvalidTime
	}

	if reason == "" {
		return fmt.Errorf("please provide a reason")
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

	adj := &model.AttendanceAdjustment{
		UserID:            userID,
		AttendanceID:      attendanceID,
		WorkDate:          workDate,
		RequestedCheckIn:  checkIn,
		RequestedCheckOut: checkOut,
		Reason:            reason,
		Status:            model.AdjustStatusPending,
	}

	if err := s.adjRepo.Create(adj); err != nil {
		log.Printf("[service][adjustment] ERROR create: user=%s date=%s err=%v", userID, workDate, err)
		return fmt.Errorf("cannot create adjustment request")
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
func (s *AttendanceAdjustmentService) ReviewRequest(reviewerID, requestID string, status model.AdjustmentStatus, note string) error {
	adj, err := s.adjRepo.FindByID(requestID)
	if err != nil {
		return ErrAdjustmentNotFound
	}

	if adj.Status != model.AdjustStatusPending {
		return fmt.Errorf("this request has already been processed")
	}

	now := time.Now()
	adj.Status = status
	adj.ReviewerID = &reviewerID
	adj.ReviewedAt = &now
	adj.ReviewerNote = note

	if err := s.adjRepo.Update(adj); err != nil {
		log.Printf("[service][adjustment] ERROR review update: id=%s err=%v", requestID, err)
		return fmt.Errorf("cannot update adjustment request")
	}

	if status == model.AdjustStatusApproved {
		s.applyAdjustment(adj)
	}

	log.Printf("[service][adjustment] reviewed: id=%s status=%s by=%s", requestID, status, reviewerID)
	return nil
}

// applyAdjustment updates the actual attendance record with the requested times.
func (s *AttendanceAdjustmentService) applyAdjustment(adj *model.AttendanceAdjustment) {
	existing, err := s.attendanceRepo.FindTodayByUserAndDate(adj.UserID, adj.WorkDate)
	if err != nil || existing == nil {
		// No attendance record exists -- create one
		// Fetch user to get branch_id via auth client
		user, err := s.authClient.GetUser(adj.UserID)
		if err != nil || user.BranchID == nil {
			log.Printf("[service][adjustment] ERROR applyAdjustment: user %s has no branch_id or fetch failed", adj.UserID)
			return
		}

		att := &model.Attendance{
			UserID:       adj.UserID,
			BranchID:     *user.BranchID,
			WorkDate:     adj.WorkDate,
			CheckInAt:    adj.RequestedCheckIn,
			CheckOutAt:   adj.RequestedCheckOut,
			Status:       model.StatusOnTime,
			Method:       "adjustment",
			IsAdjusted:   true,
			AdjustedByID: adj.ReviewerID,
			Note:         fmt.Sprintf("Attendance adjustment: %s", adj.Reason),
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
	existing.Note += fmt.Sprintf("Attendance adjustment: %s", adj.Reason)

	// Re-evaluate status if was absent/invalidated
	if existing.Status == model.StatusAbsent || existing.Status == model.StatusInvalidated {
		existing.Status = model.StatusOnTime
	}

	if err := s.attendanceRepo.Update(existing); err != nil {
		log.Printf("[service][adjustment] ERROR update attendance: %v", err)
	} else {
		log.Printf("[service][adjustment] updated attendance for %s on %s", adj.UserID, adj.WorkDate)
	}
}
