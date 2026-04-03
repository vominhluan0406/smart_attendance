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
	start, err := time.Parse("2006-01-02", leave.StartDate)
	if err != nil {
		log.Printf("[service][leave] ERROR syncAttendance parse start_date: %v", err)
		return
	}
	end, err := time.Parse("2006-01-02", leave.EndDate)
	if err != nil {
		log.Printf("[service][leave] ERROR syncAttendance parse end_date: %v", err)
		return
	}

	// Resolve branch ID from user
	var branchID string
	if leave.User != nil && leave.User.BranchID != nil {
		branchID = *leave.User.BranchID
	} else {
		// Fallback: fetch user from DB
		user, err := s.userRepo.FindByID(leave.UserID)
		if err != nil || user.BranchID == nil {
			log.Printf("[service][leave] ERROR syncAttendance: user %s has no branch_id", leave.UserID)
			return
		}
		branchID = *user.BranchID
	}

	// Resolve leave type name
	leaveTypeName := "Nghỉ phép"
	if leave.LeaveType != nil {
		leaveTypeName = leave.LeaveType.Name
	}

	log.Printf("[service][leave] syncAttendance: user=%s branch=%s dates=%s~%s type=%s",
		leave.UserID, branchID, leave.StartDate, leave.EndDate, leaveTypeName)

	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		existing, err := s.attendanceRepo.FindTodayByUserAndDate(leave.UserID, dateStr)
		if err == nil && existing != nil {
			if existing.Status == models.StatusAbsent || existing.CheckInAt == nil {
				existing.Status = models.StatusLeave
				if existing.Note != "" {
					existing.Note += " | "
				}
				existing.Note += fmt.Sprintf("Nghỉ phép: %s", leaveTypeName)
				if err := s.attendanceRepo.Update(existing); err != nil {
					log.Printf("[service][leave] ERROR syncAttendance update %s: %v", dateStr, err)
				} else {
					log.Printf("[service][leave] syncAttendance updated existing record %s -> leave", dateStr)
				}
			} else {
				log.Printf("[service][leave] syncAttendance skip %s: already has status=%s", dateStr, existing.Status)
			}
			continue
		}

		att := &models.Attendance{
			UserID:   leave.UserID,
			BranchID: branchID,
			WorkDate: dateStr,
			Status:   models.StatusLeave,
			Note:     fmt.Sprintf("Nghỉ phép: %s", leaveTypeName),
		}
		if err := s.attendanceRepo.Create(att); err != nil {
			log.Printf("[service][leave] ERROR syncAttendance create %s: %v", dateStr, err)
		} else {
			log.Printf("[service][leave] syncAttendance created leave record for %s", dateStr)
		}
	}
}

func (s *LeaveService) GetLeaveTypes() ([]models.LeaveType, error) {
	return s.leaveTypeRepo.ListAllActive()
}

func (s *LeaveService) GetRequestByID(id string) (*models.LeaveRequest, error) {
	return s.leaveRepo.FindByID(id)
}
