package repository

import (
	"github.com/smart-attendance/smart-attendance/internal/models"
	"gorm.io/gorm"
)

type PermissionRepository struct {
	db *gorm.DB
}

func NewPermissionRepository(db *gorm.DB) *PermissionRepository {
	return &PermissionRepository{db: db}
}

// FindPermissionsByRole returns all permission codes for a given role.
func (r *PermissionRepository) FindPermissionsByRole(role models.Role) ([]string, error) {
	var codes []string
	err := r.db.Raw(`
		SELECT p.code FROM role_permissions rp
		JOIN permissions p ON p.id = rp.permission_id
		WHERE rp.role = ? AND p.is_active = 1
		  AND rp.deleted_at IS NULL AND p.deleted_at IS NULL
	`, role).Scan(&codes).Error
	return codes, err
}

// FindAllPermissions returns all active permissions grouped by module.
func (r *PermissionRepository) FindAllPermissions() ([]models.Permission, error) {
	var perms []models.Permission
	err := r.db.Where("is_active = ?", true).Order("module ASC, code ASC").Find(&perms).Error
	return perms, err
}

// FindRolePermissions returns all role_permission records for a role.
func (r *PermissionRepository) FindRolePermissions(role models.Role) ([]models.RolePermission, error) {
	var rps []models.RolePermission
	err := r.db.Preload("Permission").Where("role = ?", role).Find(&rps).Error
	return rps, err
}

// SetRolePermissions replaces all permissions for a role (transactional).
func (r *PermissionRepository) SetRolePermissions(role models.Role, permissionIDs []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete existing
		if err := tx.Where("role = ?", role).Delete(&models.RolePermission{}).Error; err != nil {
			return err
		}
		// Insert new
		for _, pid := range permissionIDs {
			rp := models.RolePermission{Role: role, PermissionID: pid}
			if err := tx.Create(&rp).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
