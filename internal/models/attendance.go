package models

import "time"

type AttendanceStatus string

const (
	StatusOnTime      AttendanceStatus = "on_time"
	StatusLate        AttendanceStatus = "late"
	StatusAbsent      AttendanceStatus = "absent"
	StatusLeave       AttendanceStatus = "leave"
	StatusInvalidated AttendanceStatus = "invalidated"
)

type Attendance struct {
	BaseModel
	UserID       string           `gorm:"type:text;not null;index:idx_att_user_date" json:"user_id"`
	BranchID     string           `gorm:"type:text;not null;index:idx_att_branch_date" json:"branch_id"`
	ShiftID      *string          `gorm:"type:text;index" json:"shift_id,omitempty"`
	WorkDate     string           `gorm:"type:text;not null;index:idx_att_user_date;index:idx_att_branch_date;index:idx_att_date_status" json:"work_date"` // "2006-01-02"
	CheckInAt    *time.Time       `json:"check_in_at,omitempty"`
	CheckOutAt   *time.Time       `json:"check_out_at,omitempty"`
	Status       AttendanceStatus `gorm:"type:text;not null;default:'on_time';index:idx_att_date_status" json:"status"`
	Method       string           `gorm:"type:text" json:"method"`
	IPAddress    string           `gorm:"type:text" json:"ip_address"`
	Lat          *float64         `gorm:"type:real" json:"lat,omitempty"`
	Lng          *float64         `gorm:"type:real" json:"lng,omitempty"`
	TOTPVerified     bool             `gorm:"default:false" json:"totp_verified"`
	IPVerified       bool             `gorm:"default:false" json:"ip_verified"`
	LocVerified      bool             `gorm:"default:false" json:"loc_verified"`
	FaceVerified     bool             `gorm:"default:false" json:"face_verified"`
	NFCVerified      bool             `gorm:"default:false" json:"nfc_verified"`
	PasswordVerified bool             `gorm:"default:false" json:"password_verified"`
	BiometricVerified bool            `gorm:"default:false" json:"biometric_verified"`
	Note             string           `gorm:"type:text" json:"note,omitempty"`
	IsAdjusted       bool             `gorm:"default:false" json:"is_adjusted"`
	AdjustedByID     *string          `gorm:"type:text" json:"adjusted_by_id,omitempty"`

	// Relations (for preload)
	User   *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Branch *Branch    `gorm:"foreignKey:BranchID" json:"branch,omitempty"`
	Shift  *WorkShift `gorm:"foreignKey:ShiftID" json:"shift,omitempty"`
}
