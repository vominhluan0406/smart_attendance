package database

import (
	"fmt"
	"log"

	"github.com/smart-attendance/organization-service/internal/config"
	"github.com/smart-attendance/organization-service/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	logLevel := logger.Warn
	if cfg.Env == "development" {
		logLevel = logger.Info
	}

	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	}

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("[org][database] failed to connect: %w", err)
	}

	// Create schema if not exists and set search path
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("[org][database] failed to get sql.DB: %w", err)
	}

	_, err = sqlDB.Exec("CREATE SCHEMA IF NOT EXISTS org_service")
	if err != nil {
		return nil, fmt.Errorf("[org][database] failed to create schema: %w", err)
	}

	_, err = sqlDB.Exec("SET search_path TO org_service, public")
	if err != nil {
		return nil, fmt.Errorf("[org][database] failed to set search_path: %w", err)
	}

	// Also set search_path on the GORM session so all queries use org_service schema
	db = db.Session(&gorm.Session{})
	db.Exec("SET search_path TO org_service, public")

	log.Printf("[org][database] connected to PostgreSQL, schema=org_service")
	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	log.Printf("[org][database] running auto-migration")

	err := db.AutoMigrate(
		&model.Branch{},
		&model.BranchIPWhitelist{},
		&model.BranchLocation{},
		&model.WorkShift{},
		&model.UserShiftAssignment{},
		&model.Department{},
		&model.Holiday{},
	)
	if err != nil {
		return fmt.Errorf("[org][database] auto-migrate failed: %w", err)
	}

	log.Printf("[org][database] auto-migration completed successfully")
	return nil
}
