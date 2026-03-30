package models

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleManager  Role = "manager"
	RoleEmployee Role = "employee"
)

type User struct {
	BaseModel
	Email        string  `gorm:"type:text;uniqueIndex;not null" json:"email"`
	PasswordHash string  `gorm:"type:text;not null" json:"-"`
	FullName     string  `gorm:"type:text;not null" json:"full_name"`
	Role         Role    `gorm:"type:text;not null;default:employee" json:"role"`
	BranchID     *string `gorm:"type:text;index" json:"branch_id,omitempty"`
	IsActive     bool    `gorm:"default:true" json:"is_active"`
}
