package service

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/smart-attendance/leave-service/internal/client"
	"github.com/smart-attendance/leave-service/internal/model"
	"github.com/smart-attendance/leave-service/internal/repository"
)

var (
	ErrOverlappingLeave = errors.New("da co don nghi phep trong khoang thoi gian nay")
	ErrInvalidDateRange = errors.New("ngay ket thuc khong the truoc ngay bat dau")
	ErrLeaveNotFound    = errors.New("khong tim thay don nghi phep")
)

type LeaveService struct {
	leaveRepo        *repository.LeaveRepository
	leaveTypeRepo    *repository.LeaveTypeRepository
	authClient       *client.AuthClient
	attendanceClient *client.AttendanceClient
}

func NewLeaveService(
	leaveRepo *repository.LeaveRepository,
	leaveTypeRepo *repository.LeaveTypeRepository,
	authClient *client.AuthClient,
	attendanceClient *client.AttendanceClient,
) *LeaveService {
	return &LeaveService{
		leaveRepo:        leaveRepo,
		leaveTypeRepo:    leaveTypeRepo,
		authClient:       authClient,
		attendanceClient: attendanceClient,
	}
}

// CreateRequest validates and creates a new leave request.
func (s *LeaveService) CreateRequest(userID, branchID, leaveTypeID, startDate, endDate, reason string) error {
	// 1. Validate dates
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return fmt.Errorf("invalid start date: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return fmt.Errorf("invalid end date: %w", err)
	}

	if end.Before(start) {
		return ErrInvalidDateRange
	}

	// 2. Check for overlapping requests
	overlaps, err := s.leaveRepo.FindOverlapping(userID, startDate, endDate)
	if err != nil {
		return err
	}
	if len(overlaps) > 0 {
		return ErrOverlappingLeave
	}

	// 3. Calculate total days (simple diff for now)
	duration := end.Sub(start)
	totalDays := duration.Hours()/24 + 1

	// 4. Resolve branch if not provided
	if branchID == "" {
		user, err := s.authClient.GetUser(userID)
		if err != nil {
			log.Printf("[service][leave] ERROR resolving branch for user %s: %v", userID, err)
			return fmt.Errorf("could not resolve user branch: %w", err)
		}
		if user.BranchID != nil {
			branchID = *user.BranchID
		}
	}

	// 5. Create request
	leave := &model.LeaveRequest{
		UserID:      userID,
		BranchID:    branchID,
		LeaveTypeID: leaveTypeID,
		StartDate:   startDate,
		EndDate:     endDate,
		TotalDays:   totalDays,
		Reason:      reason,
		Status:      model.LeaveStatusPending,
	}

	log.Printf("[service][leave] create request: user=%s branch=%s type=%s dates=%s~%s days=%.1f",
		userID, branchID, leaveTypeID, startDate, endDate, totalDays)

	return s.leaveRepo.Create(leave)
}

// GetMyRequests returns paginated leave requests for a specific user.
func (s *LeaveService) GetMyRequests(userID string, page, limit int) (*repository.LeaveListResult, error) {
	return s.leaveRepo.List(repository.LeaveListParams{
		Page:   page,
		Limit:  limit,
		UserID: userID,
	})
}

// GetBranchRequests returns paginated leave requests for a specific branch,
// optionally filtered by status.
func (s *LeaveService) GetBranchRequests(branchID, status string, page, limit int) (*repository.LeaveListResult, error) {
	return s.leaveRepo.List(repository.LeaveListParams{
		Page:     page,
		Limit:    limit,
		BranchID: branchID,
		Status:   status,
	})
}

// ReviewRequest approves or rejects a leave request.
// On approval, calls attendanceClient.SyncLeave() to create attendance records.
func (s *LeaveService) ReviewRequest(reviewerID, requestID string, status model.LeaveRequestStatus, note string) error {
	leave, err := s.leaveRepo.FindByID(requestID)
	if err != nil {
		return ErrLeaveNotFound
	}

	now := time.Now()
	leave.Status = status
	leave.ReviewerID = &reviewerID
	leave.ReviewedAt = &now
	leave.ReviewerNote = note

	if err := s.leaveRepo.Update(leave); err != nil {
		return err
	}

	log.Printf("[service][leave] review: id=%s status=%s reviewer=%s", requestID, status, reviewerID)

	// If approved, sync attendance records via Attendance Service
	if status == model.LeaveStatusApproved {
		s.syncAttendance(leave)
	}

	return nil
}

// syncAttendance calls the Attendance Service to create leave attendance records.
func (s *LeaveService) syncAttendance(leave *model.LeaveRequest) {
	// Resolve leave type name for the note
	leaveTypeName := "Nghi phep"
	if leave.LeaveType != nil {
		leaveTypeName = leave.LeaveType.Name
	}

	note := fmt.Sprintf("Nghi phep: %s", leaveTypeName)

	log.Printf("[service][leave] syncAttendance via HTTP: user=%s branch=%s dates=%s~%s type=%s",
		leave.UserID, leave.BranchID, leave.StartDate, leave.EndDate, leaveTypeName)

	if err := s.attendanceClient.SyncLeave(leave.UserID, leave.BranchID, leave.StartDate, leave.EndDate, note); err != nil {
		log.Printf("[service][leave] ERROR syncAttendance: %v", err)
	}
}

// GetLeaveTypes returns all active leave types.
func (s *LeaveService) GetLeaveTypes() ([]model.LeaveType, error) {
	return s.leaveTypeRepo.ListAllActive()
}

// GetRequestByID returns a single leave request by ID.
func (s *LeaveService) GetRequestByID(id string) (*model.LeaveRequest, error) {
	return s.leaveRepo.FindByID(id)
}

// CountPendingByBranch returns the count of pending leave requests for a branch.
func (s *LeaveService) CountPendingByBranch(branchID string) (int64, error) {
	return s.leaveRepo.CountPendingByBranch(branchID)
}
