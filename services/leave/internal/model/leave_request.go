package model

import "time"

type LeaveRequestStatus string

const (
	LeaveStatusPending   LeaveRequestStatus = "pending"
	LeaveStatusApproved  LeaveRequestStatus = "approved"
	LeaveStatusRejected  LeaveRequestStatus = "rejected"
	LeaveStatusCancelled LeaveRequestStatus = "cancelled"
)

type LeaveRequest struct {
	BaseModel
	UserID       string             `gorm:"type:uuid;not null;index" json:"user_id"`
	BranchID     string             `gorm:"type:uuid;not null;index" json:"branch_id"` // denormalized for branch filtering
	LeaveTypeID  string             `gorm:"type:uuid;not null;index" json:"leave_type_id"`
	StartDate    string             `gorm:"type:text;not null" json:"start_date"` // "2006-01-02"
	EndDate      string             `gorm:"type:text;not null" json:"end_date"`
	TotalDays    float64            `gorm:"type:real;not null" json:"total_days"` // supports half-day (0.5)
	Reason       string             `gorm:"type:text" json:"reason"`
	Status       LeaveRequestStatus `gorm:"type:text;not null;default:'pending'" json:"status"`
	ReviewerID   *string            `gorm:"type:uuid" json:"reviewer_id,omitempty"`
	ReviewedAt   *time.Time         `json:"reviewed_at,omitempty"`
	ReviewerNote string             `gorm:"type:text" json:"reviewer_note,omitempty"`

	LeaveType *LeaveType `gorm:"foreignKey:LeaveTypeID" json:"leave_type,omitempty"`
}
