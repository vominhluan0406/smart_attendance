package repository

import (
	"fmt"
	"log"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/smart-attendance/auth-service/internal/model"
	"gorm.io/gorm"
)

type PermissionRepository struct {
	db    *gorm.DB
	cache *cache.Cache
}

func NewPermissionRepository(db *gorm.DB) *PermissionRepository {
	return &PermissionRepository{
		db:    db,
		cache: cache.New(5*time.Minute, 10*time.Minute),
	}
}

// FindByRoleAndCode checks if a specific permission exists for a role.
func (r *PermissionRepository) FindByRoleAndCode(role string, code string) (bool, error) {
	cacheKey := fmt.Sprintf("perm:%s:%s", role, code)
	if cached, found := r.cache.Get(cacheKey); found {
		return cached.(bool), nil
	}

	var count int64
	err := r.db.Model(&model.RolePermission{}).
		Joins("JOIN permissions ON permissions.id = role_permissions.permission_id").
		Where("role_permissions.role = ? AND permissions.code = ? AND permissions.is_active = true", role, code).
		Count(&count).Error
	if err != nil {
		log.Printf("[auth][repository][permission] FindByRoleAndCode failed: role=%s, code=%s, err=%v", role, code, err)
		return false, fmt.Errorf("check permission: %w", err)
	}

	allowed := count > 0
	r.cache.Set(cacheKey, allowed, cache.DefaultExpiration)
	return allowed, nil
}

// FindAllByRole returns all permission codes for a given role (cached).
func (r *PermissionRepository) FindAllByRole(role string) ([]string, error) {
	cacheKey := fmt.Sprintf("role_perms:%s", role)
	if cached, found := r.cache.Get(cacheKey); found {
		return cached.([]string), nil
	}

	var codes []string
	err := r.db.Model(&model.RolePermission{}).
		Select("permissions.code").
		Joins("JOIN permissions ON permissions.id = role_permissions.permission_id").
		Where("role_permissions.role = ? AND permissions.is_active = true", role).
		Pluck("permissions.code", &codes).Error
	if err != nil {
		log.Printf("[auth][repository][permission] FindAllByRole failed: role=%s, err=%v", role, err)
		return nil, fmt.Errorf("find permissions by role: %w", err)
	}

	r.cache.Set(cacheKey, codes, cache.DefaultExpiration)
	return codes, nil
}

// InvalidateCache clears all cached permission data.
func (r *PermissionRepository) InvalidateCache() {
	r.cache.Flush()
	log.Printf("[auth][repository][permission] cache invalidated")
}
