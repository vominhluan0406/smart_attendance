package model

type WorkShift struct {
	BaseModel
	BranchID             string `gorm:"type:uuid;not null;index" json:"branch_id"`
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

type UserShiftAssignment struct {
	BaseModel
	UserID        string  `gorm:"type:uuid;not null;index" json:"user_id"`
	ShiftID       string  `gorm:"type:uuid;not null;index" json:"shift_id"`
	EffectiveFrom string  `gorm:"type:text;not null" json:"effective_from"` // "2006-01-02"
	EffectiveTo   *string `gorm:"type:text" json:"effective_to,omitempty"`  // null = indefinite

	Shift *WorkShift `gorm:"foreignKey:ShiftID" json:"shift,omitempty"`
}
