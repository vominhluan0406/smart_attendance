package models

import "time"

type FraudAlertType string

const (
	FraudGPSAccuracy    FraudAlertType = "gps_accuracy"
	FraudTOTPReuse      FraudAlertType = "totp_reuse"
	FraudImpossibleTravel FraudAlertType = "impossible_travel"
	FraudNewDevice      FraudAlertType = "new_device"
	FraudIPLocationMismatch FraudAlertType = "ip_location_mismatch"
	FraudClonedAuth     FraudAlertType = "cloned_authenticator"
	FraudAnomalyTime    FraudAlertType = "anomaly_time"
	FraudConcurrentSession FraudAlertType = "concurrent_session"
)

// FraudAlert logs suspicious activity detected by anti-fraud checks.
type FraudAlert struct {
	BaseModel
	UserID      string         `gorm:"type:text;not null;index:idx_fraud_user" json:"user_id"`
	BranchID    string         `gorm:"type:text;index" json:"branch_id"`
	AlertType   FraudAlertType `gorm:"type:text;not null;index:idx_fraud_type" json:"alert_type"`
	Severity    string         `gorm:"type:text;not null;default:'warning'" json:"severity"` // warning, critical
	Description string         `gorm:"type:text" json:"description"`
	Details     string         `gorm:"type:text" json:"details"` // JSON metadata
	IPAddress   string         `gorm:"type:text" json:"ip_address"`
	Lat         *float64       `gorm:"type:real" json:"lat,omitempty"`
	Lng         *float64       `gorm:"type:real" json:"lng,omitempty"`
	IsReviewed  bool           `gorm:"default:false" json:"is_reviewed"`
	ReviewedAt  *time.Time     `json:"reviewed_at,omitempty"`
	ReviewedBy  string         `gorm:"type:text" json:"reviewed_by,omitempty"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
