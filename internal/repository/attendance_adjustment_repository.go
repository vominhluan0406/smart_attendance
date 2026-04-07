package repository

import (
	"github.com/smart-attendance/smart-attendance/internal/models"
	"gorm.io/gorm"
)

type AttendanceAdjustmentRepository struct {
	db *gorm.DB
}

func NewAttendanceAdjustmentRepository(db *gorm.DB) *AttendanceAdjustmentRepository {
	return &AttendanceAdjustmentRepository{db: db}
}

func (r *AttendanceAdjustmentRepository) Create(adj *models.AttendanceAdjustment) error {
	return r.db.Create(adj).Error
}

func (r *AttendanceAdjustmentRepository) Update(adj *models.AttendanceAdjustment) error {
	return r.db.Save(adj).Error
}

func (r *AttendanceAdjustmentRepository) FindByID(id string) (*models.AttendanceAdjustment, error) {
	var adj models.AttendanceAdjustment
	err := r.db.Preload("User").Preload("Attendance").Preload("Reviewer").First(&adj, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &adj, nil
}

type AdjustmentListParams struct {
	Page     int
	Limit    int
	UserID   string
	BranchID string
	Status   string
}

type AdjustmentListResult struct {
	Records []models.AttendanceAdjustment
	Total   int64
	Page    int
	Limit   int
}

func (r *AttendanceAdjustmentRepository) List(params AdjustmentListParams) (*AdjustmentListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 {
		params.Limit = 20
	}

	query := r.db.Model(&models.AttendanceAdjustment{})

	if params.UserID != "" {
		query = query.Where("attendance_adjustments.user_id = ?", params.UserID)
	}
	if params.Status != "" {
		query = query.Where("attendance_adjustments.status = ?", params.Status)
	}
	if params.BranchID != "" {
		query = query.Joins("JOIN users ON users.id = attendance_adjustments.user_id").
			Where("users.branch_id = ?", params.BranchID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var records []models.AttendanceAdjustment
	offset := (params.Page - 1) * params.Limit
	err := query.Preload("User").Preload("Attendance").Preload("Reviewer").
		Offset(offset).Limit(params.Limit).Order("attendance_adjustments.created_at DESC").
		Find(&records).Error

	return &AdjustmentListResult{
		Records: records,
		Total:   total,
		Page:    params.Page,
		Limit:   params.Limit,
	}, err
}

// FindPendingByUserAndDate checks if there's already a pending adjustment for the user on that date.
func (r *AttendanceAdjustmentRepository) FindPendingByUserAndDate(userID, workDate string) (*models.AttendanceAdjustment, error) {
	var adj models.AttendanceAdjustment
	err := r.db.Where("user_id = ? AND work_date = ? AND status = ?", userID, workDate, models.AdjustStatusPending).
		First(&adj).Error
	if err != nil {
		return nil, err
	}
	return &adj, nil
}
