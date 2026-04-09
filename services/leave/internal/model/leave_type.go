package model

type LeaveType struct {
	BaseModel
	Name             string `gorm:"type:text;not null" json:"name"`
	Code             string `gorm:"type:text;uniqueIndex" json:"code"` // ANNUAL, SICK, PERSONAL, etc.
	MaxDaysPerYear   int    `gorm:"default:12" json:"max_days_per_year"`
	IsPaid           bool   `gorm:"default:true" json:"is_paid"`
	RequiresApproval bool   `gorm:"default:true" json:"requires_approval"`
	Color            string `gorm:"type:text;default:'#10B981'" json:"color"`
	IsActive         bool   `gorm:"default:true" json:"is_active"`
}
