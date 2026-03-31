package models

import "time"

type AdjustmentStatus string

const (
	AdjustStatusPending  AdjustmentStatus = "pending"
	AdjustStatusApproved AdjustmentStatus = "approved"
	AdjustStatusRejected AdjustmentStatus = "rejected"
)

type AttendanceAdjustment struct {
	BaseModel
	UserID            string           `gorm:"type:text;not null;index" json:"user_id"`
	AttendanceID      *string          `gorm:"type:text" json:"attendance_id,omitempty"` // null if no record exists yet
	WorkDate          string           `gorm:"type:text;not null" json:"work_date"`      // "2006-01-02"
	RequestedCheckIn  *time.Time       `json:"requested_check_in,omitempty"`
	RequestedCheckOut *time.Time       `json:"requested_check_out,omitempty"`
	Reason            string           `gorm:"type:text;not null" json:"reason"`
	Status            AdjustmentStatus `gorm:"type:text;not null;default:'pending'" json:"status"`
	ReviewerID        *string          `gorm:"type:text" json:"reviewer_id,omitempty"`
	ReviewedAt        *time.Time       `json:"reviewed_at,omitempty"`
	ReviewerNote      string           `gorm:"type:text" json:"reviewer_note,omitempty"`

	User       *User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Attendance *Attendance `gorm:"foreignKey:AttendanceID" json:"attendance,omitempty"`
	Reviewer   *User       `gorm:"foreignKey:ReviewerID" json:"reviewer,omitempty"`
}
