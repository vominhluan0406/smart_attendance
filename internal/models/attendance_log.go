package models

import "time"

// AttendanceLog represents a single time scan (like a fingerprint reader).
// Multiple logs per user per day are allowed.
// The earliest log = check-in, the latest = check-out.
type AttendanceLog struct {
	BaseModel
	UserID       string    `gorm:"type:text;not null;index:idx_log_user_date" json:"user_id"`
	BranchID     string    `gorm:"type:text;not null" json:"branch_id"`
	ShiftID      *string   `gorm:"type:text" json:"shift_id,omitempty"`
	WorkDate     string    `gorm:"type:text;not null;index:idx_log_user_date" json:"work_date"`
	LoggedAt     time.Time `gorm:"not null" json:"logged_at"`
	Method       string    `gorm:"type:text" json:"method"`
	IPAddress    string    `gorm:"type:text" json:"ip_address"`
	Lat          *float64  `gorm:"type:real" json:"lat,omitempty"`
	Lng          *float64  `gorm:"type:real" json:"lng,omitempty"`
	TOTPVerified     bool      `gorm:"default:false" json:"totp_verified"`
	IPVerified       bool      `gorm:"default:false" json:"ip_verified"`
	LocVerified      bool      `gorm:"default:false" json:"loc_verified"`
	FaceVerified     bool      `gorm:"default:false" json:"face_verified"`
	NFCVerified      bool      `gorm:"default:false" json:"nfc_verified"`
	PasswordVerified bool      `gorm:"default:false" json:"password_verified"`
	BiometricVerified bool     `gorm:"default:false" json:"biometric_verified"`
}
