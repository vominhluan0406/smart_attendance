package repository

import (
	"github.com/smart-attendance/attendance-service/internal/model"
	"gorm.io/gorm"
)

type AttendanceAdjustmentRepository struct {
	db *gorm.DB
}

func NewAttendanceAdjustmentRepository(db *gorm.DB) *AttendanceAdjustmentRepository {
	return &AttendanceAdjustmentRepository{db: db}
}

func (r *AttendanceAdjustmentRepository) Create(adj *model.AttendanceAdjustment) error {
	return r.db.Create(adj).Error
}

func (r *AttendanceAdjustmentRepository) Update(adj *model.AttendanceAdjustment) error {
	return r.db.Save(adj).Error
}

func (r *AttendanceAdjustmentRepository) FindByID(id string) (*model.AttendanceAdjustment, error) {
	var adj model.AttendanceAdjustment
	err := r.db.First(&adj, "id = ?", id).Error
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
	Records []model.AttendanceAdjustment
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

	query := r.db.Model(&model.AttendanceAdjustment{})

	if params.UserID != "" {
		query = query.Where("user_id = ?", params.UserID)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.BranchID != "" {
		// In microservice context, we filter by branch via the user_id list
		// obtained from the org service. For now, we store branch context
		// differently — the handler should pre-filter user IDs.
		// However, if the caller passes BranchID, we can join on attendance records
		// or rely on a user_branch mapping. For simplicity, we skip this join
		// in the microservice and let the handler filter appropriately.
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var records []model.AttendanceAdjustment
	offset := (params.Page - 1) * params.Limit
	err := query.Offset(offset).Limit(params.Limit).
		Order("created_at DESC").
		Find(&records).Error

	return &AdjustmentListResult{
		Records: records,
		Total:   total,
		Page:    params.Page,
		Limit:   params.Limit,
	}, err
}

// FindPendingByUserAndDate checks if there's already a pending adjustment for the user on that date.
func (r *AttendanceAdjustmentRepository) FindPendingByUserAndDate(userID, workDate string) (*model.AttendanceAdjustment, error) {
	var adj model.AttendanceAdjustment
	err := r.db.Where("user_id = ? AND work_date = ? AND status = ?", userID, workDate, model.AdjustStatusPending).
		First(&adj).Error
	if err != nil {
		return nil, err
	}
	return &adj, nil
}
