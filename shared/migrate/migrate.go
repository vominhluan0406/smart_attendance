// Package migrate provides a simple, embeddable migration runner.
// Each service defines migrations as a list of versioned functions.
// A `schema_migrations` table tracks which versions have been applied.
//
// Usage:
//
//	migrations := []migrate.Migration{
//	    {Version: 1, Name: "create_users_table", Up: func(db *gorm.DB) error { return db.AutoMigrate(&User{}) }},
//	    {Version: 2, Name: "add_phone_column",   Up: func(db *gorm.DB) error { return db.Exec("ALTER TABLE ...").Error }},
//	}
//	migrate.Run(db, migrations)
package migrate

import (
	"log"
	"time"

	"gorm.io/gorm"
)

// SchemaMigration tracks applied migration versions.
type SchemaMigration struct {
	Version   int       `gorm:"primaryKey"`
	Name      string    `gorm:"type:varchar(255);not null"`
	AppliedAt time.Time `gorm:"not null"`
}

func (SchemaMigration) TableName() string {
	return "schema_migrations"
}

// Migration defines a single versioned migration.
type Migration struct {
	Version int
	Name    string
	Up      func(db *gorm.DB) error
}

// Run applies all unapplied migrations in order.
// Creates the schema_migrations table if it doesn't exist.
// Skips migrations that have already been applied.
func Run(db *gorm.DB, migrations []Migration) error {
	// Create migrations tracking table
	if err := db.AutoMigrate(&SchemaMigration{}); err != nil {
		return err
	}

	// Get already applied versions
	var applied []SchemaMigration
	db.Order("version ASC").Find(&applied)
	appliedSet := make(map[int]bool, len(applied))
	for _, m := range applied {
		appliedSet[m.Version] = true
	}

	// Apply pending migrations
	pending := 0
	for _, m := range migrations {
		if appliedSet[m.Version] {
			continue
		}

		log.Printf("[migrate] applying v%d: %s ...", m.Version, m.Name)

		if err := m.Up(db); err != nil {
			log.Printf("[migrate] ERROR v%d (%s): %v", m.Version, m.Name, err)
			return err
		}

		// Record migration
		record := SchemaMigration{
			Version:   m.Version,
			Name:      m.Name,
			AppliedAt: time.Now(),
		}
		if err := db.Create(&record).Error; err != nil {
			log.Printf("[migrate] ERROR recording v%d: %v", m.Version, err)
			return err
		}

		pending++
		log.Printf("[migrate] applied v%d: %s", m.Version, m.Name)
	}

	if pending == 0 {
		log.Printf("[migrate] all %d migrations already applied", len(migrations))
	} else {
		log.Printf("[migrate] applied %d new migrations (total: %d)", pending, len(migrations))
	}

	return nil
}
