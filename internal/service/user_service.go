package service

import (
	"errors"
	"fmt"

	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	userRepo *repository.UserRepository
}

func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) GetByID(id string) (*models.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (s *UserService) List(params repository.UserListParams) (*repository.UserListResult, error) {
	return s.userRepo.List(params)
}

type UpdateUserInput struct {
	FullName string      `json:"full_name"`
	Email    string      `json:"email"`
	Role     models.Role `json:"role"`
	BranchID *string     `json:"branch_id"`
	IsActive *bool       `json:"is_active"`
	Password string      `json:"password,omitempty"`
}

func (s *UserService) Update(id string, input UpdateUserInput) (*models.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if input.FullName != "" {
		user.FullName = input.FullName
	}
	if input.Email != "" && input.Email != user.Email {
		existing, err := s.userRepo.FindByEmail(input.Email)
		if err == nil && existing.ID != id {
			return nil, ErrEmailExists
		}
		user.Email = input.Email
	}
	if input.Role != "" {
		user.Role = input.Role
	}
	if input.BranchID != nil {
		user.BranchID = input.BranchID
	}
	if input.IsActive != nil {
		user.IsActive = *input.IsActive
	}
	if input.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}
		user.PasswordHash = string(hash)
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) Delete(id string) error {
	return s.userRepo.Delete(id)
}
