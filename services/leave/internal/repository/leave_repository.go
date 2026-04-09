package repository

import (
	"github.com/smart-attendance/leave-service/internal/model"
	"gorm.io/gorm"
)

type LeaveRepository struct {
	db *gorm.DB
}

func NewLeaveRepository(db *gorm.DB) *LeaveRepository {
	return &LeaveRepository{db: db}
}

func (r *LeaveRepository) Create(leave *model.LeaveRequest) error {
	return r.db.Create(leave).Error
}

func (r *LeaveRepository) Update(leave *model.LeaveRequest) error {
	return r.db.Save(leave).Error
}

func (r *LeaveRepository) FindByID(id string) (*model.LeaveRequest, error) {
	var leave model.LeaveRequest
	err := r.db.Preload("LeaveType").First(&leave, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &leave, nil
}

type LeaveListParams struct {
	Page     int
	Limit    int
	UserID   string
	BranchID string
	Status   string
}

type LeaveListResult struct {
	Records []model.LeaveRequest
	Total   int64
	Page    int
	Limit   int
}

func (r *LeaveRepository) List(params LeaveListParams) (*LeaveListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 {
		params.Limit = 20
	}

	query := r.db.Model(&model.LeaveRequest{})

	if params.UserID != "" {
		query = query.Where("user_id = ?", params.UserID)
	}
	if params.BranchID != "" {
		query = query.Where("branch_id = ?", params.BranchID)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var records []model.LeaveRequest
	offset := (params.Page - 1) * params.Limit
	err := query.Preload("LeaveType").
		Offset(offset).Limit(params.Limit).Order("created_at DESC").
		Find(&records).Error

	return &LeaveListResult{
		Records: records,
		Total:   total,
		Page:    params.Page,
		Limit:   params.Limit,
	}, err
}

// FindOverlapping finds leave requests that overlap with the given date range
// for a specific user, excluding rejected and cancelled requests.
func (r *LeaveRepository) FindOverlapping(userID, startDate, endDate string) ([]model.LeaveRequest, error) {
	var overlaps []model.LeaveRequest
	// Overlap check: (StartA <= EndB) and (EndA >= StartB)
	err := r.db.Where("user_id = ? AND status NOT IN (?)", userID,
		[]string{string(model.LeaveStatusRejected), string(model.LeaveStatusCancelled)}).
		Where("start_date <= ? AND end_date >= ?", endDate, startDate).
		Find(&overlaps).Error
	return overlaps, err
}

// CountPendingByBranch counts pending leave requests for a given branch.
func (r *LeaveRepository) CountPendingByBranch(branchID string) (int64, error) {
	var count int64
	err := r.db.Model(&model.LeaveRequest{}).
		Where("branch_id = ? AND status = ?", branchID, model.LeaveStatusPending).
		Count(&count).Error
	return count, err
}
