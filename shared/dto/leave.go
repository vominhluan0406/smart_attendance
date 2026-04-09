package dto

import "time"

type LeaveRequest struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	LeaveTypeID  string    `json:"leave_type_id"`
	StartDate    string    `json:"start_date"`
	EndDate      string    `json:"end_date"`
	TotalDays    float64   `json:"total_days"`
	Reason       string    `json:"reason"`
	Status       string    `json:"status"`
	ReviewerID   *string   `json:"reviewer_id,omitempty"`
	ReviewerNote string    `json:"reviewer_note,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UserName     string    `json:"user_name,omitempty"`
	LeaveType    string    `json:"leave_type,omitempty"`
	ReviewerName string    `json:"reviewer_name,omitempty"`
}

type LeaveType struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Code           string `json:"code,omitempty"`
	MaxDaysPerYear int    `json:"max_days_per_year"`
	IsPaid         bool   `json:"is_paid"`
	Color          string `json:"color"`
	IsActive       bool   `json:"is_active"`
}

type CreateLeaveRequest struct {
	LeaveTypeID string `json:"leave_type_id"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	Reason      string `json:"reason"`
}

type ReviewLeaveRequest struct {
	Status string `json:"status"`
	Note   string `json:"note,omitempty"`
}
