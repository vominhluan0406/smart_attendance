package dto

import "time"

type User struct {
	ID         string    `json:"id"`
	Email      string    `json:"email"`
	FullName   string    `json:"full_name"`
	Phone      string    `json:"phone,omitempty"`
	Role       string    `json:"role"`
	BranchID   *string   `json:"branch_id,omitempty"`
	BranchName string    `json:"branch_name,omitempty"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

type TokenValidation struct {
	UserID     string  `json:"user_id"`
	Email      string  `json:"email"`
	FullName   string  `json:"full_name"`
	Role       string  `json:"role"`
	BranchID   *string `json:"branch_id,omitempty"`
	BranchName string  `json:"branch_name,omitempty"`
}
