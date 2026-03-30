package repository

import (
	"github.com/smart-attendance/smart-attendance/internal/models"
	"gorm.io/gorm"
)

type BranchRepository struct {
	db *gorm.DB
}

func NewBranchRepository(db *gorm.DB) *BranchRepository {
	return &BranchRepository{db: db}
}

func (r *BranchRepository) Create(branch *models.Branch) error {
	return r.db.Create(branch).Error
}

func (r *BranchRepository) FindByID(id string) (*models.Branch, error) {
	var branch models.Branch
	if err := r.db.Preload("IPWhitelist").Preload("Locations").First(&branch, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &branch, nil
}

func (r *BranchRepository) FindByIDSimple(id string) (*models.Branch, error) {
	var branch models.Branch
	if err := r.db.First(&branch, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &branch, nil
}

type BranchListParams struct {
	Page     int
	Limit    int
	Search   string
	IsActive *bool
}

type BranchListResult struct {
	Branches []models.Branch
	Total    int64
	Page     int
	Limit    int
}

func (r *BranchRepository) List(params BranchListParams) (*BranchListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 20
	}

	query := r.db.Model(&models.Branch{})

	if params.Search != "" {
		search := "%" + params.Search + "%"
		query = query.Where("name LIKE ? OR address LIKE ?", search, search)
	}
	if params.IsActive != nil {
		query = query.Where("is_active = ?", *params.IsActive)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var branches []models.Branch
	offset := (params.Page - 1) * params.Limit
	if err := query.Offset(offset).Limit(params.Limit).Order("created_at DESC").Find(&branches).Error; err != nil {
		return nil, err
	}

	return &BranchListResult{
		Branches: branches,
		Total:    total,
		Page:     params.Page,
		Limit:    params.Limit,
	}, nil
}

func (r *BranchRepository) ListAll() ([]models.Branch, error) {
	var branches []models.Branch
	if err := r.db.Where("is_active = ?", true).Order("name ASC").Find(&branches).Error; err != nil {
		return nil, err
	}
	return branches, nil
}

func (r *BranchRepository) Update(branch *models.Branch) error {
	return r.db.Save(branch).Error
}

func (r *BranchRepository) Delete(id string) error {
	return r.db.Delete(&models.Branch{}, "id = ?", id).Error
}

// --- IP Whitelist ---

func (r *BranchRepository) ReplaceIPWhitelist(branchID string, ips []models.BranchIPWhitelist) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("branch_id = ?", branchID).Delete(&models.BranchIPWhitelist{}).Error; err != nil {
			return err
		}
		if len(ips) > 0 {
			for i := range ips {
				ips[i].BranchID = branchID
			}
			return tx.Create(&ips).Error
		}
		return nil
	})
}

// --- Locations ---

func (r *BranchRepository) ReplaceLocations(branchID string, locs []models.BranchLocation) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("branch_id = ?", branchID).Delete(&models.BranchLocation{}).Error; err != nil {
			return err
		}
		if len(locs) > 0 {
			for i := range locs {
				locs[i].BranchID = branchID
			}
			return tx.Create(&locs).Error
		}
		return nil
	})
}

// --- Employee assignment ---

func (r *BranchRepository) GetEmployeeCount(branchID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.User{}).Where("branch_id = ? AND deleted_at IS NULL", branchID).Count(&count).Error
	return count, err
}

func (r *BranchRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&models.Branch{}).Count(&count).Error
	return count, err
}
