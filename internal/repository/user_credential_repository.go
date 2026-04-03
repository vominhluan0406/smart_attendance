package repository

import (
	"github.com/smart-attendance/smart-attendance/internal/models"
	"gorm.io/gorm"
)

type UserCredentialRepository struct {
	db *gorm.DB
}

func NewUserCredentialRepository(db *gorm.DB) *UserCredentialRepository {
	return &UserCredentialRepository{db: db}
}

func (r *UserCredentialRepository) Create(c *models.UserCredential) error {
	return r.db.Create(c).Error
}

func (r *UserCredentialRepository) FindByUserID(userID string) ([]models.UserCredential, error) {
	var credentials []models.UserCredential
	err := r.db.Where("user_id = ?", userID).Find(&credentials).Error
	return credentials, err
}

func (r *UserCredentialRepository) FindByCredentialID(credID []byte) (*models.UserCredential, error) {
	var c models.UserCredential
	err := r.db.Where("credential_id = ?", credID).First(&c).Error
	return &c, err
}

func (r *UserCredentialRepository) FindByID(id string) (*models.UserCredential, error) {
	var c models.UserCredential
	err := r.db.Where("id = ?", id).First(&c).Error
	return &c, err
}

func (r *UserCredentialRepository) Delete(id string) error {
	return r.db.Delete(&models.UserCredential{}, "id = ?", id).Error
}

func (r *UserCredentialRepository) Update(c *models.UserCredential) error {
	return r.db.Save(c).Error
}
