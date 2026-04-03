package repository

import (
	"github.com/smart-attendance/smart-attendance/internal/models"
	"gorm.io/gorm"
)

type LeaveRepository struct {
	db *gorm.DB
}

func NewLeaveRepository(db *gorm.DB) *LeaveRepository {
	return &LeaveRepository{db: db}
}

func (r *LeaveRepository) Create(leave *models.LeaveRequest) error {
	return r.db.Create(leave).Error
}

func (r *LeaveRepository) Update(leave *models.LeaveRequest) error {
	return r.db.Save(leave).Error
}

func (r *LeaveRepository) FindByID(id string) (*models.LeaveRequest, error) {
	var leave models.LeaveRequest
	err := r.db.Preload("User").Preload("LeaveType").Preload("Reviewer").First(&leave, "id = ?", id).Error
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
	Records []models.LeaveRequest
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

	query := r.db.Model(&models.LeaveRequest{})

	if params.UserID != "" {
		query = query.Where("user_id = ?", params.UserID)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}

	if params.BranchID != "" {
		query = query.Joins("JOIN users ON users.id = leave_requests.user_id").
			Where("users.branch_id = ?", params.BranchID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var records []models.LeaveRequest
	offset := (params.Page - 1) * params.Limit
	err := query.Preload("User").Preload("LeaveType").Preload("Reviewer").
		Offset(offset).Limit(params.Limit).Order("created_at DESC").
		Find(&records).Error

	return &LeaveListResult{
		Records: records,
		Total:   total,
		Page:    params.Page,
		Limit:   params.Limit,
	}, err
}

func (r *LeaveRepository) FindOverlapping(userID, startDate, endDate string) ([]models.LeaveRequest, error) {
	var overlaps []models.LeaveRequest
	// Overlap check: (StartA <= EndB) and (EndA >= StartB)
	// And status not rejected/cancelled
	err := r.db.Where("user_id = ? AND status NOT IN (?)", userID, []string{string(models.LeaveStatusRejected), string(models.LeaveStatusCancelled)}).
		Where("start_date <= ? AND end_date >= ?", endDate, startDate).
		Find(&overlaps).Error
	return overlaps, err
}
