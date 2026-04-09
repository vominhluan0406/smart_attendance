package model

import "time"

// UserSession tracks active JWT sessions per user for concurrent session detection.
type UserSession struct {
	BaseModel
	UserID       string    `gorm:"type:uuid;not null;index:idx_session_user" json:"user_id"`
	TokenHash    string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"token_hash"`
	IPAddress    string    `gorm:"type:varchar(45)" json:"ip_address"`
	UserAgent    string    `gorm:"type:text" json:"user_agent"`
	ExpiresAt    time.Time `gorm:"not null" json:"expires_at"`
	IsRevoked    bool      `gorm:"default:false" json:"is_revoked"`
	LastActiveAt time.Time `json:"last_active_at"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (UserSession) TableName() string {
	return "user_sessions"
}
