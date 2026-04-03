package repository

import (
	"github.com/smart-attendance/smart-attendance/internal/models"
	"gorm.io/gorm"
)

type UserDeviceRepository struct {
	db *gorm.DB
}

func NewUserDeviceRepository(db *gorm.DB) *UserDeviceRepository {
	return &UserDeviceRepository{db: db}
}

func (r *UserDeviceRepository) Create(d *models.UserDevice) error {
	return r.db.Create(d).Error
}

func (r *UserDeviceRepository) Update(d *models.UserDevice) error {
	return r.db.Save(d).Error
}

func (r *UserDeviceRepository) FindByUserAndFingerprint(userID, hash string) (*models.UserDevice, error) {
	var d models.UserDevice
	err := r.db.Where("user_id = ? AND fingerprint_hash = ?", userID, hash).First(&d).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *UserDeviceRepository) FindByUserID(userID string) ([]models.UserDevice, error) {
	var devices []models.UserDevice
	err := r.db.Where("user_id = ?", userID).Order("last_seen_at DESC").Find(&devices).Error
	return devices, err
}

func (r *UserDeviceRepository) CountByUserID(userID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.UserDevice{}).Where("user_id = ? AND is_blocked = false", userID).Count(&count).Error
	return count, err
}
