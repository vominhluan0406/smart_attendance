package database

import (
	"log"
	"time"

	"github.com/smart-attendance/auth-service/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Seed seeds permissions, role-permissions, and a default admin user.
func Seed(db *gorm.DB) error {
	// Silence logging during seeding to avoid massive SQL dumps in development mode
	originalLogger := db.Logger
	db.Logger = originalLogger.LogMode(logger.Warn)
	defer func() { db.Logger = originalLogger }()

	return db.Transaction(func(tx *gorm.DB) error {
		seedPermissions(tx)

		// Seed default admin user if no users exist
		var userCount int64
		if err := tx.Model(&model.User{}).Count(&userCount).Error; err != nil {
			log.Printf("[auth][seed] ERROR: count users failed: %v", err)
			return err
		}

		if userCount == 0 {
			hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
			if err != nil {
				log.Printf("[auth][seed] ERROR: bcrypt hash failed: %v", err)
				return err
			}

			joinDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.Local)
			admin := &model.User{
				EmployeeCode: "ADM001",
				Email:        "admin@smartattendance.com",
				PasswordHash: string(hash),
				FullName:     "System Administrator",
				Phone:        "0901000001",
				Role:         model.RoleAdmin,
				Position:     "System Administrator",
				JoinDate:     &joinDate,
				IsActive:     true,
			}

			if err := tx.Create(admin).Error; err != nil {
				log.Printf("[auth][seed] ERROR: failed to create admin user: %v", err)
				return err
			}

			log.Printf("[auth][seed] created default admin: admin@smartattendance.com / password123")
		} else {
			log.Printf("[auth][seed] users table has %d rows, skipping admin seed", userCount)
		}

		return nil
	})
}

func seedPermissions(db *gorm.DB) {
	var count int64
	db.Model(&model.Permission{}).Count(&count)
	if count == 0 {
		// Fresh DB — create all permissions
		perms := model.DefaultPermissions()
		if err := db.Create(&perms).Error; err != nil {
			log.Printf("[auth][seed] warning: failed to create permissions: %v", err)
			return
		}
		log.Printf("[auth][seed] created %d permissions", len(perms))

		// Build code → ID map
		permMap := make(map[string]string, len(perms))
		for _, p := range perms {
			permMap[p.Code] = p.ID
		}

		// Create role-permission mappings
		rolePerms := model.DefaultRolePermissions()
		var rps []model.RolePermission
		for role, codes := range rolePerms {
			for _, code := range codes {
				if pid, ok := permMap[code]; ok {
					rps = append(rps, model.RolePermission{Role: role, PermissionID: pid})
				}
			}
		}

		if err := db.Create(&rps).Error; err != nil {
			log.Printf("[auth][seed] warning: failed to create role permissions: %v", err)
			return
		}
		log.Printf("[auth][seed] created %d role-permission mappings", len(rps))
	}

	// Ensure any new permissions added to code are created in DB
	ensureNewPermissions(db)

	// Always ensure every role in DefaultRolePermissions has its mappings
	ensureRolePermissions(db)
}

// ensureNewPermissions creates any permissions defined in code but missing from the DB.
func ensureNewPermissions(db *gorm.DB) {
	var existing []model.Permission
	db.Find(&existing)
	existingCodes := make(map[string]bool, len(existing))
	for _, p := range existing {
		existingCodes[p.Code] = true
	}

	var added int
	for _, p := range model.DefaultPermissions() {
		if existingCodes[p.Code] {
			continue
		}
		if err := db.Create(&p).Error; err == nil {
			added++
		}
	}
	if added > 0 {
		log.Printf("[auth][seed] created %d new permissions", added)
	}
}

// ensureRolePermissions checks each role's expected permissions and inserts any missing ones.
func ensureRolePermissions(db *gorm.DB) {
	// Build code → ID map from DB
	var allPerms []model.Permission
	db.Find(&allPerms)
	permMap := make(map[string]string, len(allPerms))
	for _, p := range allPerms {
		permMap[p.Code] = p.ID
	}

	rolePerms := model.DefaultRolePermissions()
	for role, codes := range rolePerms {
		// Get existing permission IDs for this role
		var existingPermIDs []string
		db.Model(&model.RolePermission{}).
			Where("role = ?", role).
			Pluck("permission_id", &existingPermIDs)

		existingSet := make(map[string]bool, len(existingPermIDs))
		for _, pid := range existingPermIDs {
			existingSet[pid] = true
		}

		// Insert missing
		var added int
		for _, code := range codes {
			pid, ok := permMap[code]
			if !ok {
				continue
			}
			if existingSet[pid] {
				continue
			}
			rp := model.RolePermission{Role: role, PermissionID: pid}
			if err := db.Create(&rp).Error; err == nil {
				added++
			}
		}
		if added > 0 {
			log.Printf("[auth][seed] added %d missing permissions for role %s", added, role)
		}
	}
}
