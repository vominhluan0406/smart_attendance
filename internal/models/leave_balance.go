package models

type LeaveBalance struct {
	BaseModel
	UserID      string  `gorm:"type:text;not null;uniqueIndex:idx_lb_unique" json:"user_id"`
	LeaveTypeID string  `gorm:"type:text;not null;uniqueIndex:idx_lb_unique" json:"leave_type_id"`
	Year        int     `gorm:"not null;uniqueIndex:idx_lb_unique" json:"year"`
	TotalDays   float64 `gorm:"type:real;default:12" json:"total_days"`
	UsedDays    float64 `gorm:"type:real;default:0" json:"used_days"`

	User      *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	LeaveType *LeaveType `gorm:"foreignKey:LeaveTypeID" json:"leave_type,omitempty"`
}

func (lb *LeaveBalance) RemainingDays() float64 {
	return lb.TotalDays - lb.UsedDays
}
