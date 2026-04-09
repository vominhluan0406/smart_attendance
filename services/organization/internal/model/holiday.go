package model

type Holiday struct {
	BaseModel
	Name        string  `gorm:"type:text;not null" json:"name"`
	Date        string  `gorm:"type:text;not null;index" json:"date"` // "2006-01-02"
	BranchID    *string `gorm:"type:uuid;index" json:"branch_id,omitempty"` // null = company-wide
	HolidayType string  `gorm:"type:text;default:'company'" json:"holiday_type"` // national, company, branch
	IsRecurring bool    `gorm:"default:false" json:"is_recurring"`
	IsActive    bool    `gorm:"default:true" json:"is_active"`

	Branch *Branch `gorm:"foreignKey:BranchID" json:"branch,omitempty"`
}
