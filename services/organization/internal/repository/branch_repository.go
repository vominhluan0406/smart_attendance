package repository

import (
	"github.com/smart-attendance/organization-service/internal/model"
	"gorm.io/gorm"
)

type BranchRepository struct {
	db *gorm.DB
}

func NewBranchRepository(db *gorm.DB) *BranchRepository {
	return &BranchRepository{db: db}
}

// Create creates a new branch.
func (r *BranchRepository) Create(branch *model.Branch) error {
	return r.db.Create(branch).Error
}

// FindByID returns a branch with IP whitelist and locations preloaded.
func (r *BranchRepository) FindByID(id string) (*model.Branch, error) {
	var branch model.Branch
	if err := r.db.Preload("IPWhitelist").Preload("Locations").First(&branch, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &branch, nil
}

// FindByIDSimple returns a branch without preloading relations (quick lookup).
func (r *BranchRepository) FindByIDSimple(id string) (*model.Branch, error) {
	var branch model.Branch
	if err := r.db.First(&branch, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &branch, nil
}

// Update saves updates to a branch.
func (r *BranchRepository) Update(branch *model.Branch) error {
	return r.db.Save(branch).Error
}

// Delete soft-deletes a branch by ID.
func (r *BranchRepository) Delete(id string) error {
	return r.db.Delete(&model.Branch{}, "id = ?", id).Error
}

// ListAll returns all active branches ordered by name.
func (r *BranchRepository) ListAll() ([]model.Branch, error) {
	var branches []model.Branch
	if err := r.db.Where("is_active = ?", true).Order("name ASC").Find(&branches).Error; err != nil {
		return nil, err
	}
	return branches, nil
}

// BranchListParams holds pagination and filter parameters for branch listing.
type BranchListParams struct {
	Page     int
	Limit    int
	Search   string
	IsActive *bool
}

// BranchListResult contains the paginated result.
type BranchListResult struct {
	Branches []model.Branch
	Total    int64
	Page     int
	Limit    int
}

// List returns a paginated list of branches with optional filters.
func (r *BranchRepository) List(params BranchListParams) (*BranchListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 20
	}

	query := r.db.Model(&model.Branch{})

	if params.Search != "" {
		search := "%" + params.Search + "%"
		query = query.Where("name ILIKE ? OR address ILIKE ?", search, search)
	}
	if params.IsActive != nil {
		query = query.Where("is_active = ?", *params.IsActive)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	var branches []model.Branch
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

// ReplaceIPWhitelist replaces all IP whitelist entries for a branch.
func (r *BranchRepository) ReplaceIPWhitelist(branchID string, ips []model.BranchIPWhitelist) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("branch_id = ?", branchID).Delete(&model.BranchIPWhitelist{}).Error; err != nil {
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

// ReplaceLocations replaces all location entries for a branch.
func (r *BranchRepository) ReplaceLocations(branchID string, locs []model.BranchLocation) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("branch_id = ?", branchID).Delete(&model.BranchLocation{}).Error; err != nil {
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
