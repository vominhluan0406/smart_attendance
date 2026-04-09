package repository

import (
	"github.com/smart-attendance/auth-service/internal/model"
	"gorm.io/gorm"
)

type CredentialRepository struct {
	db *gorm.DB
}

func NewCredentialRepository(db *gorm.DB) *CredentialRepository {
	return &CredentialRepository{db: db}
}

func (r *CredentialRepository) Create(cred *model.UserCredential) error {
	return r.db.Create(cred).Error
}

func (r *CredentialRepository) Update(cred *model.UserCredential) error {
	return r.db.Save(cred).Error
}

func (r *CredentialRepository) FindByID(id string) (*model.UserCredential, error) {
	var cred model.UserCredential
	err := r.db.First(&cred, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &cred, nil
}

func (r *CredentialRepository) FindByCredentialID(credentialID []byte) (*model.UserCredential, error) {
	var cred model.UserCredential
	err := r.db.First(&cred, "credential_id = ?", credentialID).Error
	if err != nil {
		return nil, err
	}
	return &cred, nil
}

func (r *CredentialRepository) ListByUserID(userID string) ([]model.UserCredential, error) {
	var creds []model.UserCredential
	err := r.db.Find(&creds, "user_id = ?", userID).Error
	return creds, err
}

func (r *CredentialRepository) Delete(id string) error {
	return r.db.Delete(&model.UserCredential{}, "id = ?", id).Error
}
