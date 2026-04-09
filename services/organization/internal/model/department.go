package model

type Department struct {
	BaseModel
	BranchID  string  `gorm:"type:uuid;not null;index" json:"branch_id"`
	Name      string  `gorm:"type:text;not null" json:"name"`
	Code      string  `gorm:"type:text" json:"code"`
	ManagerID *string `gorm:"type:uuid" json:"manager_id,omitempty"`
	IsActive  bool    `gorm:"default:true" json:"is_active"`

	Branch *Branch `gorm:"foreignKey:BranchID" json:"branch,omitempty"`
}
