package model

import "time"

type AdjustmentStatus string

const (
	AdjustStatusPending  AdjustmentStatus = "pending"
	AdjustStatusApproved AdjustmentStatus = "approved"
	AdjustStatusRejected AdjustmentStatus = "rejected"
)

type AttendanceAdjustment struct {
	BaseModel
	UserID            string           `gorm:"type:uuid;not null;index" json:"user_id"`
	AttendanceID      *string          `gorm:"type:uuid" json:"attendance_id,omitempty"`
	WorkDate          string           `gorm:"type:text;not null" json:"work_date"`
	RequestedCheckIn  *time.Time       `json:"requested_check_in,omitempty"`
	RequestedCheckOut *time.Time       `json:"requested_check_out,omitempty"`
	Reason            string           `gorm:"type:text;not null" json:"reason"`
	Status            AdjustmentStatus `gorm:"type:text;not null;default:'pending'" json:"status"`
	ReviewerID        *string          `gorm:"type:uuid" json:"reviewer_id,omitempty"`
	ReviewedAt        *time.Time       `json:"reviewed_at,omitempty"`
	ReviewerNote      string           `gorm:"type:text" json:"reviewer_note,omitempty"`
}
