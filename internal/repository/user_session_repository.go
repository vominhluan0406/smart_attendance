package repository

import (
	"time"

	"github.com/smart-attendance/smart-attendance/internal/models"
	"gorm.io/gorm"
)

type UserSessionRepository struct {
	db *gorm.DB
}

func NewUserSessionRepository(db *gorm.DB) *UserSessionRepository {
	return &UserSessionRepository{db: db}
}

func (r *UserSessionRepository) Create(s *models.UserSession) error {
	return r.db.Create(s).Error
}

func (r *UserSessionRepository) FindByTokenHash(hash string) (*models.UserSession, error) {
	var s models.UserSession
	err := r.db.Where("token_hash = ? AND is_revoked = false AND expires_at > ?", hash, time.Now()).First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *UserSessionRepository) CountActiveSessions(userID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.UserSession{}).
		Where("user_id = ? AND is_revoked = false AND expires_at > ?", userID, time.Now()).
		Count(&count).Error
	return count, err
}

func (r *UserSessionRepository) FindActiveSessions(userID string) ([]models.UserSession, error) {
	var sessions []models.UserSession
	err := r.db.Where("user_id = ? AND is_revoked = false AND expires_at > ?", userID, time.Now()).
		Order("last_active_at DESC").Find(&sessions).Error
	return sessions, err
}

func (r *UserSessionRepository) RevokeOldest(userID string) error {
	var oldest models.UserSession
	err := r.db.Where("user_id = ? AND is_revoked = false AND expires_at > ?", userID, time.Now()).
		Order("last_active_at ASC").First(&oldest).Error
	if err != nil {
		return err
	}
	return r.db.Model(&oldest).Update("is_revoked", true).Error
}

func (r *UserSessionRepository) RevokeByTokenHash(hash string) error {
	return r.db.Model(&models.UserSession{}).Where("token_hash = ?", hash).Update("is_revoked", true).Error
}

func (r *UserSessionRepository) RevokeAllForUser(userID string) error {
	return r.db.Model(&models.UserSession{}).Where("user_id = ? AND is_revoked = false", userID).Update("is_revoked", true).Error
}

func (r *UserSessionRepository) UpdateLastActive(hash string) error {
	return r.db.Model(&models.UserSession{}).Where("token_hash = ?", hash).Update("last_active_at", time.Now()).Error
}

func (r *UserSessionRepository) CleanExpired() error {
	return r.db.Where("expires_at < ? OR is_revoked = true", time.Now().Add(-24*time.Hour)).Delete(&models.UserSession{}).Error
}
