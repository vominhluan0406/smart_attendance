package model

import "time"

// UserDevice tracks known devices for each user (device fingerprinting).
type UserDevice struct {
	BaseModel
	UserID          string    `gorm:"type:uuid;not null;index:idx_device_user" json:"user_id"`
	FingerprintHash string    `gorm:"type:text;not null;index:idx_device_fp" json:"fingerprint_hash"`
	UserAgent       string    `gorm:"type:text" json:"user_agent"`
	DeviceName      string    `gorm:"type:text" json:"device_name"`
	LastSeenAt      time.Time `json:"last_seen_at"`
	IsTrusted       bool      `gorm:"default:false" json:"is_trusted"`
	IsBlocked       bool      `gorm:"default:false" json:"is_blocked"`
}
