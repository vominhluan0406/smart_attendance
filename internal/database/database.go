package database

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"github.com/smart-attendance/smart-attendance/internal/config"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	// Ensure data directory exists
	dir := filepath.Dir(cfg.DBPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	logLevel := logger.Warn
	if cfg.Env == "development" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Enable WAL mode for concurrent read performance
	if err := db.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}

	// SQLite performance tuning
	db.Exec("PRAGMA synchronous=NORMAL")
	db.Exec("PRAGMA cache_size=-20000") // 20MB cache
	db.Exec("PRAGMA busy_timeout=5000")
	db.Exec("PRAGMA foreign_keys=ON")

	return db, nil
}

func AutoMigrate(db *gorm.DB, models ...interface{}) error {
	return db.AutoMigrate(models...)
}
