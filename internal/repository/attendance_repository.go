package repository

import (
	"time"

	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/timezone"
	"gorm.io/gorm"
)

type AttendanceRepository struct {
	db *gorm.DB
}

func NewAttendanceRepository(db *gorm.DB) *AttendanceRepository {
	return &AttendanceRepository{db: db}
}

func (r *AttendanceRepository) Create(att *models.Attendance) error {
	return r.db.Create(att).Error
}

func (r *AttendanceRepository) Update(att *models.Attendance) error {
	return r.db.Save(att).Error
}

func (r *AttendanceRepository) FindByID(id string) (*models.Attendance, error) {
	var att models.Attendance
	if err := r.db.Preload("User").Preload("Branch").First(&att, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &att, nil
}

// FindTodayByUser returns today's attendance record for a user (if any).
// Uses composite index (user_id, work_date) for fast lookup.
func (r *AttendanceRepository) FindTodayByUser(userID string) (*models.Attendance, error) {
	today := timezone.Now().Format("2006-01-02")

	var att models.Attendance
	err := r.db.Where("user_id = ? AND work_date = ?", userID, today).
		First(&att).Error
	if err != nil {
		return nil, err
	}
	return &att, nil
}

type AttendanceListParams struct {
	Page     int
	Limit    int
	UserID   string
	BranchID string
	Status   string
	DateFrom *time.Time
	DateTo   *time.Time
}

type AttendanceListResult struct {
	Records []models.Attendance
	Total   int64
	Page    int
	Limit   int
}

func (r *AttendanceRepository) List(params AttendanceListParams) (*AttendanceListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 20
	}

	query := r.db.Model(&models.Attendance{})

	if params.UserID != "" {
		query = query.Where("user_id = ?", params.UserID)
	}
	if params.BranchID != "" {
		query = query.Where("branch_id = ?", params.BranchID)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.DateFrom != nil {
		query = query.Where("work_date >= ?", params.DateFrom.Format("2006-01-02"))
	}
	if params.DateTo != nil {
		query = query.Where("work_date <= ?", params.DateTo.Format("2006-01-02"))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var records []models.Attendance
	offset := (params.Page - 1) * params.Limit
	if err := query.Preload("User").Preload("Branch").
		Offset(offset).Limit(params.Limit).Order("check_in_at DESC").
		Find(&records).Error; err != nil {
		return nil, err
	}

	return &AttendanceListResult{
		Records: records,
		Total:   total,
		Page:    params.Page,
		Limit:   params.Limit,
	}, nil
}
