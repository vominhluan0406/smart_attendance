package repository

import (
	"fmt"
	"log"

	"github.com/smart-attendance/auth-service/internal/model"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(user *model.User) error {
	if err := r.db.Create(user).Error; err != nil {
		log.Printf("[auth][repository][user] create failed: email=%s, err=%v", user.Email, err)
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *UserRepository) FindByID(id string) (*model.User, error) {
	var user model.User
	if err := r.db.Where("id = ?", id).First(&user).Error; err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) Update(user *model.User) error {
	if err := r.db.Save(user).Error; err != nil {
		log.Printf("[auth][repository][user] update failed: id=%s, err=%v", user.ID, err)
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (r *UserRepository) Delete(id string) error {
	if err := r.db.Where("id = ?", id).Delete(&model.User{}).Error; err != nil {
		log.Printf("[auth][repository][user] delete failed: id=%s, err=%v", id, err)
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

type UserListParams struct {
	Page     int
	Limit    int
	BranchID string
	Role     string
	Search   string
	IsActive *bool
}

type UserListResult struct {
	Users []model.User
	Total int64
}

func (r *UserRepository) List(params UserListParams) (*UserListResult, error) {
	query := r.db.Model(&model.User{})

	if params.BranchID != "" {
		query = query.Where("branch_id = ?", params.BranchID)
	}
	if params.Role != "" {
		query = query.Where("role = ?", params.Role)
	}
	if params.Search != "" {
		search := "%" + params.Search + "%"
		query = query.Where("full_name ILIKE ? OR email ILIKE ? OR employee_code ILIKE ?", search, search, search)
	}
	if params.IsActive != nil {
		query = query.Where("is_active = ?", *params.IsActive)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Printf("[auth][repository][user] list count failed: %v", err)
		return nil, fmt.Errorf("count users: %w", err)
	}

	var users []model.User
	offset := (params.Page - 1) * params.Limit
	if err := query.Order("created_at DESC").Offset(offset).Limit(params.Limit).Find(&users).Error; err != nil {
		log.Printf("[auth][repository][user] list query failed: %v", err)
		return nil, fmt.Errorf("list users: %w", err)
	}

	return &UserListResult{Users: users, Total: total}, nil
}
