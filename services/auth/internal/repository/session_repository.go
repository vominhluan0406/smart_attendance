package repository

import (
	"fmt"
	"log"
	"time"

	"github.com/smart-attendance/auth-service/internal/model"
	"gorm.io/gorm"
)

type SessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(session *model.UserSession) error {
	if err := r.db.Create(session).Error; err != nil {
		log.Printf("[auth][repository][session] create failed: user_id=%s, err=%v", session.UserID, err)
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (r *SessionRepository) FindByToken(tokenHash string) (*model.UserSession, error) {
	var session model.UserSession
	if err := r.db.Where("token_hash = ? AND is_revoked = false AND expires_at > ?", tokenHash, time.Now()).First(&session).Error; err != nil {
		return nil, fmt.Errorf("find session by token: %w", err)
	}
	return &session, nil
}

func (r *SessionRepository) RevokeByUser(userID string) error {
	if err := r.db.Model(&model.UserSession{}).
		Where("user_id = ? AND is_revoked = false", userID).
		Update("is_revoked", true).Error; err != nil {
		log.Printf("[auth][repository][session] revoke by user failed: user_id=%s, err=%v", userID, err)
		return fmt.Errorf("revoke sessions by user: %w", err)
	}
	return nil
}

func (r *SessionRepository) RevokeByToken(tokenHash string) error {
	if err := r.db.Model(&model.UserSession{}).
		Where("token_hash = ?", tokenHash).
		Update("is_revoked", true).Error; err != nil {
		log.Printf("[auth][repository][session] revoke by token failed: err=%v", err)
		return fmt.Errorf("revoke session by token: %w", err)
	}
	return nil
}

func (r *SessionRepository) CountActive(userID string) (int64, error) {
	var count int64
	if err := r.db.Model(&model.UserSession{}).
		Where("user_id = ? AND is_revoked = false AND expires_at > ?", userID, time.Now()).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count active sessions: %w", err)
	}
	return count, nil
}

// RevokeOldest revokes the oldest active session for a user.
func (r *SessionRepository) RevokeOldest(userID string) error {
	var oldest model.UserSession
	if err := r.db.Where("user_id = ? AND is_revoked = false AND expires_at > ?", userID, time.Now()).
		Order("created_at ASC").
		First(&oldest).Error; err != nil {
		return fmt.Errorf("find oldest session: %w", err)
	}

	if err := r.db.Model(&oldest).Update("is_revoked", true).Error; err != nil {
		log.Printf("[auth][repository][session] revoke oldest failed: user_id=%s, session_id=%s, err=%v", userID, oldest.ID, err)
		return fmt.Errorf("revoke oldest session: %w", err)
	}

	log.Printf("[auth][repository][session] revoked oldest session: user_id=%s, session_id=%s", userID, oldest.ID)
	return nil
}

func (r *SessionRepository) UpdateLastActive(tokenHash string) error {
	return r.db.Model(&model.UserSession{}).
		Where("token_hash = ?", tokenHash).
		Update("last_active_at", time.Now()).Error
}
