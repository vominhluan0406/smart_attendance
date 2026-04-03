package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/repository"
	"github.com/smart-attendance/smart-attendance/internal/timezone"
)

var (
	ErrOverlappingLeave = errors.New("đã có đơn nghỉ phép trong khoảng thời gian này")
	ErrInvalidDateRange = errors.New("ngày kết thúc không thể trước ngày bắt đầu")
	ErrLeaveNotFound    = errors.New("không tìm thấy đơn nghỉ phép")
)

type LeaveService struct {
	leaveRepo      *repository.LeaveRepository
	leaveTypeRepo  *repository.LeaveTypeRepository
	attendanceRepo *repository.AttendanceRepository
	userRepo       *repository.UserRepository
}

func NewLeaveService(
	leaveRepo *repository.LeaveRepository,
	leaveTypeRepo *repository.LeaveTypeRepository,
	attendanceRepo *repository.AttendanceRepository,
	userRepo *repository.UserRepository,
) *LeaveService {
	return &LeaveService{
		leaveRepo:      leaveRepo,
		leaveTypeRepo:  leaveTypeRepo,
		attendanceRepo: attendanceRepo,
		userRepo:       userRepo,
	}
}

func (s *LeaveService) CreateRequest(userID, leaveTypeID, startDate, endDate, reason string) error {
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

	// 4. Create request
	leave := &models.LeaveRequest{
		UserID:      userID,
		LeaveTypeID: leaveTypeID,
		StartDate:   startDate,
		EndDate:     endDate,
		TotalDays:   totalDays,
		Reason:      reason,
		Status:      models.LeaveStatusPending,
	}

	return s.leaveRepo.Create(leave)
}

func (s *LeaveService) GetMyRequests(userID string, page, limit int) (*repository.LeaveListResult, error) {
	return s.leaveRepo.List(repository.LeaveListParams{
		Page:   page,
		Limit:  limit,
		UserID: userID,
	})
}

func (s *LeaveService) GetBranchRequests(branchID, status string, page, limit int) (*repository.LeaveListResult, error) {
	return s.leaveRepo.List(repository.LeaveListParams{
		Page:     page,
		Limit:    limit,
		BranchID: branchID,
		Status:   status,
	})
}

func (s *LeaveService) ReviewRequest(reviewerID, requestID string, status models.LeaveRequestStatus, note string) error {
	leave, err := s.leaveRepo.FindByID(requestID)
	if err != nil {
		return ErrLeaveNotFound
	}

	now := timezone.Now()
	leave.Status = status
	leave.ReviewerID = &reviewerID
	leave.ReviewedAt = &now
	leave.ReviewerNote = note

	if err := s.leaveRepo.Update(leave); err != nil {
		return err
	}

	// If approved, create/update attendance records
	if status == models.LeaveStatusApproved {
		s.syncAttendance(leave)
	}

	return nil
}

func (s *LeaveService) syncAttendance(leave *models.LeaveRequest) {
	start, _ := time.Parse("2006-01-02", leave.StartDate)
	end, _ := time.Parse("2006-01-02", leave.EndDate)

	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		// Use a temporary attendance record to check/create
		existing, err := s.attendanceRepo.FindTodayByUserAndDate(leave.UserID, dateStr)
		if err == nil && existing != nil {
			// If already has check-in, maybe keep it but add note?
			// For now, if it's "absent" or empty, change to "leave"
			if existing.Status == models.StatusAbsent || existing.CheckInAt == nil {
				existing.Status = models.StatusLeave
				if existing.Note != "" {
					existing.Note += " | "
				}
				existing.Note += fmt.Sprintf("Nghỉ phép: %s", leave.LeaveType.Name)
				s.attendanceRepo.Update(existing)
			}
			continue
		}

		// Create new leave attendance record
		att := &models.Attendance{
			UserID:   leave.UserID,
			BranchID: *leave.User.BranchID,
			WorkDate: dateStr,
			Status:   models.StatusLeave,
			Note:     fmt.Sprintf("Nghỉ phép: %s", leave.LeaveType.Name),
		}
		s.attendanceRepo.Create(att)
	}
}

func (s *LeaveService) GetLeaveTypes() ([]models.LeaveType, error) {
	return s.leaveTypeRepo.ListAllActive()
}

func (s *LeaveService) GetRequestByID(id string) (*models.LeaveRequest, error) {
	return s.leaveRepo.FindByID(id)
}
