package database

import (
	"fmt"
	"log"

	"github.com/smart-attendance/smart-attendance/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// RunMigrations performs data migrations after AutoMigrate has created/updated tables.
// These are idempotent — safe to run multiple times.
func RunMigrations(db *gorm.DB) error {
	if err := migrateBackfillWorkDate(db); err != nil {
		return err
	}
	if err := migrateCreateDefaultShifts(db); err != nil {
		return err
	}
	if err := migrateManagerDevicePermissions(db); err != nil {
		return err
	}
	if err := migrateCreateManagerDeviceUsers(db); err != nil {
		return err
	}
	return nil
}

// migrateBackfillWorkDate populates work_date from check_in_at for existing attendance records.
func migrateBackfillWorkDate(db *gorm.DB) error {
	result := db.Exec(`
		UPDATE attendances
		SET work_date = strftime('%Y-%m-%d', check_in_at)
		WHERE work_date = '' OR work_date IS NULL
	`)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Printf("[migration] backfilled work_date for %d attendance records", result.RowsAffected)
	}
	return nil
}

// migrateCreateDefaultShifts creates a default work shift for each branch
// that doesn't have one yet, based on the branch's WorkStartTime/WorkEndTime.
func migrateCreateDefaultShifts(db *gorm.DB) error {
	// Find branches that have no shifts
	type branchInfo struct {
		ID            string
		Name          string
		WorkStartTime string
		WorkEndTime   string
	}

	var branches []branchInfo
	err := db.Raw(`
		SELECT b.id, b.name, b.work_start_time, b.work_end_time
		FROM branches b
		LEFT JOIN work_shifts ws ON ws.branch_id = b.id AND ws.deleted_at IS NULL
		WHERE b.deleted_at IS NULL
		AND ws.id IS NULL
	`).Scan(&branches).Error
	if err != nil {
		return err
	}

	if len(branches) == 0 {
		return nil
	}

	for _, b := range branches {
		startTime := b.WorkStartTime
		if startTime == "" {
			startTime = "08:00"
		}
		endTime := b.WorkEndTime
		if endTime == "" {
			endTime = "17:00"
		}

		err := db.Exec(`
			INSERT INTO work_shifts (id, branch_id, name, code, start_time, end_time, grace_period_minutes, late_threshold_minutes, is_overnight, break_duration_minutes, working_days, color, is_default, is_active, created_at, updated_at)
			VALUES (lower(hex(randomblob(4)) || '-' || hex(randomblob(2)) || '-4' || substr(hex(randomblob(2)),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(hex(randomblob(2)),2) || '-' || hex(randomblob(6))),
				?, 'Ca chính', 'DEFAULT', ?, ?, 15, 0, 0, 60, '1,2,3,4,5', '#3B82F6', 1, 1, datetime('now'), datetime('now'))
		`, b.ID, startTime, endTime).Error
		if err != nil {
			log.Printf("[migration] warning: failed to create default shift for branch %s: %v", b.Name, err)
			continue
		}
		log.Printf("[migration] created default shift for branch: %s (%s-%s)", b.Name, startTime, endTime)
	}

	return nil
}

// migrateManagerDevicePermissions adds role_permissions for the new manager_device role
// on existing databases that already have permissions seeded.
func migrateManagerDevicePermissions(db *gorm.DB) error {
	// Check if permissions table has data at all
	var permCount int64
	db.Model(&models.Permission{}).Count(&permCount)
	if permCount == 0 {
		return nil // Fresh DB, seedPermissions will handle it
	}

	// Get expected permission codes for manager_device
	roleCodes := models.DefaultRolePermissions()[models.RoleManagerDevice]
	if len(roleCodes) == 0 {
		return nil
	}

	// Find which permissions are missing for this role
	var existingCodes []string
	db.Raw(`
		SELECT p.code FROM role_permissions rp
		JOIN permissions p ON p.id = rp.permission_id
		WHERE rp.role = ?
	`, models.RoleManagerDevice).Scan(&existingCodes)

	existingSet := make(map[string]bool, len(existingCodes))
	for _, c := range existingCodes {
		existingSet[c] = true
	}

	var missingCodes []string
	for _, code := range roleCodes {
		if !existingSet[code] {
			missingCodes = append(missingCodes, code)
		}
	}

	log.Printf("[migration] manager_device: expected=%d existing=%d missing=%d codes=%v",
		len(roleCodes), len(existingCodes), len(missingCodes), missingCodes)

	if len(missingCodes) == 0 {
		return nil
	}

	// Find permission IDs by code — use Unscoped to avoid soft-delete filter on Permission
	var perms []models.Permission
	if err := db.Where("code IN ? AND is_active = ?", missingCodes, true).Find(&perms).Error; err != nil {
		log.Printf("[migration] warning: failed to find permissions: %v", err)
		return err
	}
	log.Printf("[migration] found %d permission records for missing codes", len(perms))

	var rps []models.RolePermission
	for _, p := range perms {
		rps = append(rps, models.RolePermission{Role: models.RoleManagerDevice, PermissionID: p.ID})
	}

	if len(rps) > 0 {
		if err := db.Create(&rps).Error; err != nil {
			log.Printf("[migration] warning: failed to create manager_device permissions: %v", err)
			return nil
		}
		log.Printf("[migration] created %d role-permission mappings for manager_device", len(rps))
	} else {
		log.Printf("[migration] WARNING: no permission records found for codes %v — check permissions table", missingCodes)
	}

	return nil
}

// migrateCreateManagerDeviceUsers creates manager_device (kiosk) accounts
// for each branch that doesn't already have one.
func migrateCreateManagerDeviceUsers(db *gorm.DB) error {
	// Check if any manager_device users exist
	var count int64
	db.Model(&models.User{}).Where("role = ?", models.RoleManagerDevice).Count(&count)
	if count > 0 {
		return nil // Already has device accounts
	}

	// Get all branches
	type branchInfo struct {
		ID   string
		Name string
	}
	var branches []branchInfo
	if err := db.Raw("SELECT id, name FROM branches WHERE deleted_at IS NULL").Scan(&branches).Error; err != nil {
		return nil
	}
	if len(branches) == 0 {
		return nil
	}

	pw, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		return nil
	}

	for i, b := range branches {
		email := fmt.Sprintf("device.%d@demo.com", i+1)
		user := &models.User{
			EmployeeCode: fmt.Sprintf("DEV%03d", i+1),
			Email:        email,
			PasswordHash: string(pw),
			FullName:     fmt.Sprintf("Kiosk %s", b.Name),
			Role:         models.RoleManagerDevice,
			BranchID:     &b.ID,
			Position:     "Kiosk Device",
			IsActive:     true,
		}
		if err := db.Create(user).Error; err != nil {
			log.Printf("[migration] warning: failed to create device user for branch %s: %v", b.Name, err)
			continue
		}
		log.Printf("[migration] created manager_device user: %s (branch: %s)", email, b.Name)
	}

	return nil
}
