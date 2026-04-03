package repository

import (
	"github.com/smart-attendance/smart-attendance/internal/models"
	"gorm.io/gorm"
)

type LeaveTypeRepository struct {
	db *gorm.DB
}

func NewLeaveTypeRepository(db *gorm.DB) *LeaveTypeRepository {
	return &LeaveTypeRepository{db: db}
}

func (r *LeaveTypeRepository) ListAllActive() ([]models.LeaveType, error) {
	var types []models.LeaveType
	err := r.db.Where("is_active = ?", true).Order("name ASC").Find(&types).Error
	return types, err
}

func (r *LeaveTypeRepository) FindByID(id string) (*models.LeaveType, error) {
	var lt models.LeaveType
	if err := r.db.First(&lt, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &lt, nil
}
