package models

type WorkShift struct {
	BaseModel
	BranchID             string `gorm:"type:text;not null;index" json:"branch_id"`
	Name                 string `gorm:"type:text;not null" json:"name"`
	Code                 string `gorm:"type:text" json:"code"`
	StartTime            string `gorm:"type:text;not null" json:"start_time"`
	EndTime              string `gorm:"type:text;not null" json:"end_time"`
	GracePeriodMinutes   int    `gorm:"default:15" json:"grace_period_minutes"`
	LateThresholdMinutes int    `gorm:"default:0" json:"late_threshold_minutes"`
	IsOvernight          bool   `gorm:"default:false" json:"is_overnight"`
	BreakDurationMinutes int    `gorm:"default:60" json:"break_duration_minutes"`
	WorkingDays          string `gorm:"type:text;default:'1,2,3,4,5'" json:"working_days"` // 1=Mon,...,7=Sun
	Color                string `gorm:"type:text;default:'#3B82F6'" json:"color"`
	IsDefault            bool   `gorm:"default:false" json:"is_default"`
	IsActive             bool   `gorm:"default:true" json:"is_active"`

	Branch *Branch `gorm:"foreignKey:BranchID" json:"branch,omitempty"`
}
