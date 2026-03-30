package models

type UserShiftAssignment struct {
	BaseModel
	UserID        string  `gorm:"type:text;not null;index" json:"user_id"`
	ShiftID       string  `gorm:"type:text;not null;index" json:"shift_id"`
	EffectiveFrom string  `gorm:"type:text;not null" json:"effective_from"` // "2006-01-02"
	EffectiveTo   *string `gorm:"type:text" json:"effective_to,omitempty"` // null = indefinite

	User  *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Shift *WorkShift `gorm:"foreignKey:ShiftID" json:"shift,omitempty"`
}
