package repository

import (
	"time"

	"github.com/smart-attendance/attendance-service/internal/model"
	"gorm.io/gorm"
)

type FraudAlertRepository struct {
	db *gorm.DB
}

func NewFraudAlertRepository(db *gorm.DB) *FraudAlertRepository {
	return &FraudAlertRepository{db: db}
}

func (r *FraudAlertRepository) Create(a *model.FraudAlert) error {
	return r.db.Create(a).Error
}

func (r *FraudAlertRepository) CountUnreviewed() (int64, error) {
	var count int64
	err := r.db.Model(&model.FraudAlert{}).Where("is_reviewed = false").Count(&count).Error
	return count, err
}

// FraudAlertListResult holds paginated fraud alert results.
type FraudAlertListResult struct {
	Records []model.FraudAlert
	Total   int64
	Page    int
	Limit   int
}

// List returns paginated fraud alerts with optional filters.
func (r *FraudAlertRepository) List(branchID string, alertType string, severity string, isReviewed *bool, dateFrom, dateTo *time.Time, page, limit int) (*FraudAlertListResult, error) {
	q := r.db.Model(&model.FraudAlert{})

	if branchID != "" {
		q = q.Where("branch_id = ?", branchID)
	}
	if alertType != "" {
		q = q.Where("alert_type = ?", alertType)
	}
	if severity != "" {
		q = q.Where("severity = ?", severity)
	}
	if isReviewed != nil {
		q = q.Where("is_reviewed = ?", *isReviewed)
	}
	if dateFrom != nil {
		q = q.Where("created_at >= ?", *dateFrom)
	}
	if dateTo != nil {
		q = q.Where("created_at <= ?", *dateTo)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}

	var records []model.FraudAlert
	offset := (page - 1) * limit
	if err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&records).Error; err != nil {
		return nil, err
	}

	return &FraudAlertListResult{
		Records: records,
		Total:   total,
		Page:    page,
		Limit:   limit,
	}, nil
}

// FindByID returns a single fraud alert by ID.
func (r *FraudAlertRepository) FindByID(id string) (*model.FraudAlert, error) {
	var alert model.FraudAlert
	if err := r.db.First(&alert, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &alert, nil
}

// MarkReviewed marks a fraud alert as reviewed.
func (r *FraudAlertRepository) MarkReviewed(id, reviewedBy string) error {
	now := time.Now()
	return r.db.Model(&model.FraudAlert{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_reviewed": true,
		"reviewed_at": now,
		"reviewed_by": reviewedBy,
	}).Error
}
