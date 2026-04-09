package model

import "time"

type Role string

const (
	RoleAdmin         Role = "admin"
	RoleManager       Role = "manager"
	RoleManagerDevice Role = "manager_device"
	RoleEmployee      Role = "employee"
)

type User struct {
	BaseModel
	EmployeeCode string     `gorm:"type:varchar(50);index" json:"employee_code,omitempty"`
	Email        string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string     `gorm:"type:text" json:"-"`
	FullName     string     `gorm:"type:varchar(255);not null" json:"full_name"`
	Phone        string     `gorm:"type:varchar(20)" json:"phone,omitempty"`
	Role         Role       `gorm:"type:varchar(50);not null;default:employee" json:"role"`
	BranchID     *string    `gorm:"type:uuid;index" json:"branch_id,omitempty"`
	DepartmentID *string    `gorm:"type:uuid;index" json:"department_id,omitempty"`
	Position     string     `gorm:"type:varchar(100)" json:"position,omitempty"`
	JoinDate     *time.Time `json:"join_date,omitempty"`
	IsActive     bool       `gorm:"default:true" json:"is_active"`

	// OAuth fields (kept for compatibility)
	OAuthProvider string `gorm:"type:varchar(50)" json:"oauth_provider,omitempty"`
	OAuthID       string `gorm:"type:varchar(255);index" json:"oauth_id,omitempty"`

	// Associations
	Credentials []UserCredential `gorm:"foreignKey:UserID" json:"-"`
}

func (User) TableName() string {
	return "users"
}
