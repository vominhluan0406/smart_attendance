package dto

import "time"

type Attendance struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	BranchID    string     `json:"branch_id"`
	WorkDate    string     `json:"work_date"`
	CheckInAt   *time.Time `json:"check_in_at,omitempty"`
	CheckOutAt  *time.Time `json:"check_out_at,omitempty"`
	Status      string     `json:"status"`
	Method      string     `json:"method"`
	Note        string     `json:"note,omitempty"`
	IsAdjusted  bool       `json:"is_adjusted"`
	UserName    string     `json:"user_name,omitempty"`
	BranchName  string     `json:"branch_name,omitempty"`
}

type LogTimeRequest struct {
	TOTPCode          string   `json:"totp_code,omitempty"`
	ScannedBranchID   string   `json:"scanned_branch_id,omitempty"`
	Lat               *float64 `json:"lat,omitempty"`
	Lng               *float64 `json:"lng,omitempty"`
	AccuracyM         *float64 `json:"accuracy_m,omitempty"`
	DeviceFingerprint string   `json:"device_fingerprint,omitempty"`
	PasswordVerified  bool     `json:"password_verified,omitempty"`
	BiometricVerified bool     `json:"biometric_verified,omitempty"`
}

type LogTimeResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	CheckInAt    string `json:"check_in_at,omitempty"`
	CheckOutAt   string `json:"check_out_at,omitempty"`
	Status       string `json:"status"`
	Method       string `json:"method"`
	LogCount     int    `json:"log_count"`
	NewDevice    bool   `json:"new_device,omitempty"`
	AnomalyFlag  bool   `json:"anomaly_flag,omitempty"`
}

type FraudAlert struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	BranchID    string    `json:"branch_id"`
	AlertType   string    `json:"alert_type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	IPAddress   string    `json:"ip_address,omitempty"`
	IsReviewed  bool      `json:"is_reviewed"`
	CreatedAt   time.Time `json:"created_at"`
	UserName    string    `json:"user_name,omitempty"`
	UserEmail   string    `json:"user_email,omitempty"`
}

type AdjustmentRequest struct {
	WorkDate  string `json:"work_date"`
	CheckIn   string `json:"check_in,omitempty"`
	CheckOut  string `json:"check_out,omitempty"`
	Reason    string `json:"reason"`
}

type AdjustmentReview struct {
	Status string `json:"status"`
	Note   string `json:"note,omitempty"`
}

type SyncLeaveRequest struct {
	UserID   string `json:"user_id"`
	BranchID string `json:"branch_id"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Note      string `json:"note"`
}
