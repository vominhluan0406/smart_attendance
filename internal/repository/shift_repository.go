package repository

import (
	"github.com/smart-attendance/smart-attendance/internal/models"
	"gorm.io/gorm"
)

type ShiftRepository struct {
	db *gorm.DB
}

func NewShiftRepository(db *gorm.DB) *ShiftRepository {
	return &ShiftRepository{db: db}
}

// FindUserShift resolves the active work shift for a user on a given date.
// Priority: 1) UserShiftAssignment for the date, 2) Branch default shift.
func (r *ShiftRepository) FindUserShift(userID string, branchID string, date string) (*models.WorkShift, error) {
	// 1. Check user-specific shift assignment
	var assignment models.UserShiftAssignment
	err := r.db.
		Where("user_id = ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to >= ?)", userID, date, date).
		Order("effective_from DESC").
		First(&assignment).Error
	if err == nil {
		var shift models.WorkShift
		if err := r.db.First(&shift, "id = ? AND is_active = ?", assignment.ShiftID, true).Error; err == nil {
			return &shift, nil
		}
	}

	// 2. Fallback to branch default shift
	var defaultShift models.WorkShift
	err = r.db.
		Where("branch_id = ? AND is_default = ? AND is_active = ?", branchID, true, true).
		First(&defaultShift).Error
	if err == nil {
		return &defaultShift, nil
	}

	// 3. No shift found
	return nil, gorm.ErrRecordNotFound
}

// FindByBranch returns all active shifts for a branch.
func (r *ShiftRepository) FindByBranch(branchID string) ([]models.WorkShift, error) {
	var shifts []models.WorkShift
	err := r.db.Where("branch_id = ? AND is_active = ?", branchID, true).
		Order("is_default DESC, name ASC").
		Find(&shifts).Error
	return shifts, err
}

// Create creates a new work shift.
func (r *ShiftRepository) Create(shift *models.WorkShift) error {
	return r.db.Create(shift).Error
}

// Update updates an existing work shift.
func (r *ShiftRepository) Update(shift *models.WorkShift) error {
	return r.db.Save(shift).Error
}
