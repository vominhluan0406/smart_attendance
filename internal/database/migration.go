package database

import (
	"log"

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
