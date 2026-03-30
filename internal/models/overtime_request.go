package models

import "time"

type OvertimeStatus string

const (
	OTStatusPending  OvertimeStatus = "pending"
	OTStatusApproved OvertimeStatus = "approved"
	OTStatusRejected OvertimeStatus = "rejected"
)

type OvertimeRequest struct {
	BaseModel
	UserID       string         `gorm:"type:text;not null;index" json:"user_id"`
	WorkDate     string         `gorm:"type:text;not null" json:"work_date"` // "2006-01-02"
	PlannedStart string         `gorm:"type:text;not null" json:"planned_start"` // "18:00"
	PlannedEnd   string         `gorm:"type:text;not null" json:"planned_end"`   // "21:00"
	PlannedHours float64        `gorm:"type:real" json:"planned_hours"`
	Reason       string         `gorm:"type:text" json:"reason"`
	Status       OvertimeStatus `gorm:"type:text;not null;default:'pending'" json:"status"`
	ReviewerID   *string        `gorm:"type:text" json:"reviewer_id,omitempty"`
	ReviewedAt   *time.Time     `json:"reviewed_at,omitempty"`
	ReviewerNote string         `gorm:"type:text" json:"reviewer_note,omitempty"`

	User     *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Reviewer *User `gorm:"foreignKey:ReviewerID" json:"reviewer,omitempty"`
}
