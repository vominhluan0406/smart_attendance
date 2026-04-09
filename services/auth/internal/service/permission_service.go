package service

import (
	"log"

	"github.com/smart-attendance/auth-service/internal/repository"
)

type PermissionService struct {
	permRepo *repository.PermissionRepository
}

func NewPermissionService(permRepo *repository.PermissionRepository) *PermissionService {
	return &PermissionService{permRepo: permRepo}
}

// HasPermission checks if a role has a specific permission (cached).
func (s *PermissionService) HasPermission(role, code string) (bool, error) {
	allowed, err := s.permRepo.FindByRoleAndCode(role, code)
	if err != nil {
		log.Printf("[auth][service][permission] HasPermission failed: role=%s, code=%s, err=%v", role, code, err)
		return false, err
	}
	return allowed, nil
}

// GetRolePermissions returns all permission codes for a role (cached).
func (s *PermissionService) GetRolePermissions(role string) ([]string, error) {
	codes, err := s.permRepo.FindAllByRole(role)
	if err != nil {
		log.Printf("[auth][service][permission] GetRolePermissions failed: role=%s, err=%v", role, err)
		return nil, err
	}
	return codes, nil
}
