package repository

import (
	"github.com/smart-attendance/smart-attendance/internal/models"
	"gorm.io/gorm"
)

type FraudAlertRepository struct {
	db *gorm.DB
}

func NewFraudAlertRepository(db *gorm.DB) *FraudAlertRepository {
	return &FraudAlertRepository{db: db}
}

func (r *FraudAlertRepository) Create(a *models.FraudAlert) error {
	return r.db.Create(a).Error
}

func (r *FraudAlertRepository) CountUnreviewed() (int64, error) {
	var count int64
	err := r.db.Model(&models.FraudAlert{}).Where("is_reviewed = false").Count(&count).Error
	return count, err
}
