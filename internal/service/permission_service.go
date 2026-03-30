package service

import (
	"fmt"
	"log"
	"time"

	"github.com/smart-attendance/smart-attendance/internal/cache"
	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/repository"
)

const permCacheTTL = 10 * time.Minute

type PermissionService struct {
	permRepo *repository.PermissionRepository
	cache    *cache.Cache
}

func NewPermissionService(permRepo *repository.PermissionRepository, cache *cache.Cache) *PermissionService {
	return &PermissionService{permRepo: permRepo, cache: cache}
}

// HasPermission checks if a role has a specific permission code.
// Results are cached per role for 10 minutes.
func (s *PermissionService) HasPermission(role models.Role, permCode string) bool {
	perms := s.getPermissionsForRole(role)
	_, ok := perms[permCode]
	return ok
}

// HasAnyPermission checks if a role has at least one of the given permission codes.
func (s *PermissionService) HasAnyPermission(role models.Role, permCodes ...string) bool {
	perms := s.getPermissionsForRole(role)
	for _, code := range permCodes {
		if _, ok := perms[code]; ok {
			return true
		}
	}
	return false
}

// GetPermissionsForRole returns all permission codes for a role (cached).
func (s *PermissionService) GetPermissionsForRole(role models.Role) []string {
	perms := s.getPermissionsForRole(role)
	codes := make([]string, 0, len(perms))
	for code := range perms {
		codes = append(codes, code)
	}
	return codes
}

// GetAllPermissions returns all active permissions (for admin UI).
func (s *PermissionService) GetAllPermissions() ([]models.Permission, error) {
	return s.permRepo.FindAllPermissions()
}

// GetRolePermissions returns role_permission records for a role (for admin UI).
func (s *PermissionService) GetRolePermissions(role models.Role) ([]models.RolePermission, error) {
	return s.permRepo.FindRolePermissions(role)
}

// UpdateRolePermissions replaces permissions for a role and invalidates cache.
func (s *PermissionService) UpdateRolePermissions(role models.Role, permissionIDs []string) error {
	if err := s.permRepo.SetRolePermissions(role, permissionIDs); err != nil {
		return err
	}
	s.invalidateRole(role)
	log.Printf("[service][permission] updated permissions for role %s (%d permissions)", role, len(permissionIDs))
	return nil
}

// InvalidateAll clears all permission caches.
func (s *PermissionService) InvalidateAll() {
	s.cache.Delete(s.cacheKey(models.RoleAdmin))
	s.cache.Delete(s.cacheKey(models.RoleManager))
	s.cache.Delete(s.cacheKey(models.RoleEmployee))
}

// --- internal ---

func (s *PermissionService) getPermissionsForRole(role models.Role) map[string]struct{} {
	key := s.cacheKey(role)

	if cached, ok := s.cache.Get(key); ok {
		return cached.(map[string]struct{})
	}

	codes, err := s.permRepo.FindPermissionsByRole(role)
	if err != nil {
		log.Printf("[service][permission] error loading permissions for role %s: %v", role, err)
		return map[string]struct{}{}
	}

	permSet := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		permSet[code] = struct{}{}
	}

	s.cache.Set(key, permSet, permCacheTTL)
	return permSet
}

func (s *PermissionService) invalidateRole(role models.Role) {
	s.cache.Delete(s.cacheKey(role))
}

func (s *PermissionService) cacheKey(role models.Role) string {
	return fmt.Sprintf("perms:%s", role)
}
