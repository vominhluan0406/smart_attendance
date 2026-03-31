package repository

import (
	"github.com/smart-attendance/smart-attendance/internal/models"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) FindByID(id string) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, "email = ?", email).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

type UserListParams struct {
	Page     int
	Limit    int
	Search   string
	Role     string
	BranchID string
	IsActive *bool
}

type UserListResult struct {
	Users []models.User
	Total int64
	Page  int
	Limit int
}

func (r *UserRepository) List(params UserListParams) (*UserListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 20
	}

	query := r.db.Model(&models.User{})

	if params.Search != "" {
		search := "%" + params.Search + "%"
		query = query.Where("full_name LIKE ? OR email LIKE ?", search, search)
	}
	if params.Role != "" {
		query = query.Where("role = ?", params.Role)
	}
	if params.BranchID != "" {
		query = query.Where("branch_id = ?", params.BranchID)
	}
	if params.IsActive != nil {
		query = query.Where("is_active = ?", *params.IsActive)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var users []models.User
	offset := (params.Page - 1) * params.Limit
	if err := query.Offset(offset).Limit(params.Limit).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, err
	}

	return &UserListResult{
		Users: users,
		Total: total,
		Page:  params.Page,
		Limit: params.Limit,
	}, nil
}

func (r *UserRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

func (r *UserRepository) Delete(id string) error {
	return r.db.Delete(&models.User{}, "id = ?", id).Error
}

func (r *UserRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&models.User{}).Count(&count).Error
	return count, err
}

func (r *UserRepository) FindByOAuthID(provider, oauthID string) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, "oauth_provider = ? AND oauth_id = ?", provider, oauthID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
