package repository

import (
	"github.com/smart-attendance/attendance-service/internal/model"
	"gorm.io/gorm"
)

type UserDeviceRepository struct {
	db *gorm.DB
}

func NewUserDeviceRepository(db *gorm.DB) *UserDeviceRepository {
	return &UserDeviceRepository{db: db}
}

func (r *UserDeviceRepository) Create(d *model.UserDevice) error {
	return r.db.Create(d).Error
}

func (r *UserDeviceRepository) Update(d *model.UserDevice) error {
	return r.db.Save(d).Error
}

func (r *UserDeviceRepository) FindByUserAndFingerprint(userID, hash string) (*model.UserDevice, error) {
	var d model.UserDevice
	err := r.db.Where("user_id = ? AND fingerprint_hash = ?", userID, hash).First(&d).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}
