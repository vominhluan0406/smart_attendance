package event

import "time"

// Subjects for NATS pub/sub.
const (
	SubjectUserUpdated   = "user.updated"
	SubjectUserDeleted   = "user.deleted"
	SubjectBranchUpdated = "branch.updated"
	SubjectBranchDeleted = "branch.deleted"
	SubjectLeaveApproved = "leave.approved"
	SubjectFraudDetected = "fraud.detected"
	SubjectAttendanceCreated = "attendance.created"
)

// UserEvent is published when a user is created/updated/deleted.
type UserEvent struct {
	UserID    string    `json:"user_id"`
	Action    string    `json:"action"` // created, updated, deleted
	Timestamp time.Time `json:"timestamp"`
}

// BranchEvent is published when a branch config changes.
type BranchEvent struct {
	BranchID  string    `json:"branch_id"`
	Action    string    `json:"action"` // updated, deleted
	Timestamp time.Time `json:"timestamp"`
}

// LeaveApprovedEvent is published when a leave request is approved.
type LeaveApprovedEvent struct {
	RequestID string    `json:"request_id"`
	UserID    string    `json:"user_id"`
	BranchID  string    `json:"branch_id"`
	StartDate string    `json:"start_date"`
	EndDate   string    `json:"end_date"`
	Note      string    `json:"note"`
	Timestamp time.Time `json:"timestamp"`
}

// FraudDetectedEvent is published when fraud is detected.
type FraudDetectedEvent struct {
	AlertID   string    `json:"alert_id"`
	UserID    string    `json:"user_id"`
	BranchID  string    `json:"branch_id"`
	AlertType string    `json:"alert_type"`
	Severity  string    `json:"severity"`
	Timestamp time.Time `json:"timestamp"`
}
