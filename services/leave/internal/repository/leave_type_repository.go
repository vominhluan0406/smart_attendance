package repository

import (
	"github.com/smart-attendance/leave-service/internal/model"
	"gorm.io/gorm"
)

type LeaveTypeRepository struct {
	db *gorm.DB
}

func NewLeaveTypeRepository(db *gorm.DB) *LeaveTypeRepository {
	return &LeaveTypeRepository{db: db}
}

func (r *LeaveTypeRepository) ListAllActive() ([]model.LeaveType, error) {
	var types []model.LeaveType
	err := r.db.Where("is_active = ?", true).Order("name ASC").Find(&types).Error
	return types, err
}

func (r *LeaveTypeRepository) FindByID(id string) (*model.LeaveType, error) {
	var lt model.LeaveType
	if err := r.db.First(&lt, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &lt, nil
}
