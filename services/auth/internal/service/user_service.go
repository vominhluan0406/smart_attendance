package service

import (
	"errors"
	"fmt"
	"log"

	"time"

	"github.com/smart-attendance/auth-service/internal/model"
	"github.com/smart-attendance/auth-service/internal/repository"
	"github.com/smart-attendance/shared/event"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	userRepo *repository.UserRepository
	eventBus *event.Bus
}

func NewUserService(userRepo *repository.UserRepository, eventBus *event.Bus) *UserService {
	return &UserService{userRepo: userRepo, eventBus: eventBus}
}

func (s *UserService) GetByID(id string) (*model.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found: %s", id)
		}
		log.Printf("[auth][service][user] GetByID failed: id=%s, err=%v", id, err)
		return nil, err
	}
	return user, nil
}

func (s *UserService) GetByEmail(email string) (*model.User, error) {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found: %s", email)
		}
		log.Printf("[auth][service][user] GetByEmail failed: email=%s, err=%v", email, err)
		return nil, err
	}
	return user, nil
}

type CreateUserInput struct {
	EmployeeCode string  `json:"employee_code"`
	Email        string  `json:"email"`
	Password     string  `json:"password"`
	FullName     string  `json:"full_name"`
	Phone        string  `json:"phone"`
	Role         string  `json:"role"`
	BranchID     *string `json:"branch_id"`
	DepartmentID *string `json:"department_id"`
	Position     string  `json:"position"`
}

func (s *UserService) Create(input CreateUserInput) (*model.User, error) {
	// Check for duplicate email
	existing, _ := s.userRepo.FindByEmail(input.Email)
	if existing != nil {
		log.Printf("[auth][service][user] create failed: duplicate email=%s", input.Email)
		return nil, errors.New("email already exists")
	}

	// Validate role
	role := model.Role(input.Role)
	if role != model.RoleAdmin && role != model.RoleManager && role != model.RoleManagerDevice && role != model.RoleEmployee {
		return nil, fmt.Errorf("invalid role: %s", input.Role)
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[auth][service][user] create failed: bcrypt error, email=%s, err=%v", input.Email, err)
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		EmployeeCode: input.EmployeeCode,
		Email:        input.Email,
		PasswordHash: string(hash),
		FullName:     input.FullName,
		Phone:        input.Phone,
		Role:         role,
		BranchID:     input.BranchID,
		DepartmentID: input.DepartmentID,
		Position:     input.Position,
		IsActive:     true,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	log.Printf("[auth][service][user] created: id=%s, email=%s, role=%s", user.ID, user.Email, user.Role)
	s.publishUserEvent(user.ID, "created")
	return user, nil
}

type UpdateUserInput struct {
	EmployeeCode *string `json:"employee_code,omitempty"`
	Email        *string `json:"email,omitempty"`
	Password     *string `json:"password,omitempty"`
	FullName     *string `json:"full_name,omitempty"`
	Phone        *string `json:"phone,omitempty"`
	Role         *string `json:"role,omitempty"`
	BranchID     *string `json:"branch_id,omitempty"`
	DepartmentID *string `json:"department_id,omitempty"`
	Position     *string `json:"position,omitempty"`
	IsActive     *bool   `json:"is_active,omitempty"`
}

func (s *UserService) Update(id string, input UpdateUserInput) (*model.User, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %s", id)
	}

	if input.EmployeeCode != nil {
		user.EmployeeCode = *input.EmployeeCode
	}
	if input.Email != nil {
		// Check for duplicate email
		existing, _ := s.userRepo.FindByEmail(*input.Email)
		if existing != nil && existing.ID != id {
			return nil, errors.New("email already exists")
		}
		user.Email = *input.Email
	}
	if input.Password != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(*input.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("hash password: %w", err)
		}
		user.PasswordHash = string(hash)
	}
	if input.FullName != nil {
		user.FullName = *input.FullName
	}
	if input.Phone != nil {
		user.Phone = *input.Phone
	}
	if input.Role != nil {
		role := model.Role(*input.Role)
		if role != model.RoleAdmin && role != model.RoleManager && role != model.RoleManagerDevice && role != model.RoleEmployee {
			return nil, fmt.Errorf("invalid role: %s", *input.Role)
		}
		user.Role = role
	}
	if input.BranchID != nil {
		user.BranchID = input.BranchID
	}
	if input.DepartmentID != nil {
		user.DepartmentID = input.DepartmentID
	}
	if input.Position != nil {
		user.Position = *input.Position
	}
	if input.IsActive != nil {
		user.IsActive = *input.IsActive
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	log.Printf("[auth][service][user] updated: id=%s, email=%s", user.ID, user.Email)
	s.publishUserEvent(user.ID, "updated")
	return user, nil
}

func (s *UserService) Delete(id string) error {
	if err := s.userRepo.Delete(id); err != nil {
		return err
	}
	log.Printf("[auth][service][user] deleted: id=%s", id)
	s.publishUserEvent(id, "deleted")
	return nil
}

func (s *UserService) publishUserEvent(userID, action string) {
	if s.eventBus == nil {
		return
	}
	subject := event.SubjectUserUpdated
	if action == "deleted" {
		subject = event.SubjectUserDeleted
	}
	s.eventBus.Publish(subject, event.UserEvent{
		UserID:    userID,
		Action:    action,
		Timestamp: time.Now(),
	})
}

func (s *UserService) List(params repository.UserListParams) (*repository.UserListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 20
	}

	result, err := s.userRepo.List(params)
	if err != nil {
		log.Printf("[auth][service][user] list failed: err=%v", err)
		return nil, err
	}
	return result, nil
}
