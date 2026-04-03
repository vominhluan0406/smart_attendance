package models

import "time"

type UserStreak struct {
	BaseModel
	UserID       string    `gorm:"type:text;not null;uniqueIndex" json:"user_id"`
	CurrentCount int       `gorm:"default:0" json:"current_count"`
	MaxCount     int       `gorm:"default:0" json:"max_count"`
	LastCheckIn  time.Time `json:"last_check_in"`
}

type BadgeType string

const (
	BadgeEarlyBird     BadgeType = "early_bird"
	BadgePerfectWeek   BadgeType = "perfect_week"
	BadgePerfectMonth  BadgeType = "perfect_month"
	BadgePunctual      BadgeType = "punctual"
)

type UserBadge struct {
	BaseModel
	UserID    string    `gorm:"type:text;not null;index" json:"user_id"`
	BadgeType BadgeType `gorm:"type:text;not null" json:"badge_type"`
	EarnedAt  time.Time `json:"earned_at"`
	Reason    string    `gorm:"type:text" json:"reason"`
}
