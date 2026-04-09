package database

import (
	"fmt"
	"log"

	"github.com/smart-attendance/leave-service/internal/config"
	"github.com/smart-attendance/leave-service/internal/model"
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
		return nil, fmt.Errorf("[leave][database] failed to connect: %w", err)
	}

	// Create schema if not exists and set search path
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("[leave][database] failed to get sql.DB: %w", err)
	}

	_, err = sqlDB.Exec("CREATE SCHEMA IF NOT EXISTS leave_service")
	if err != nil {
		return nil, fmt.Errorf("[leave][database] failed to create schema: %w", err)
	}

	_, err = sqlDB.Exec("SET search_path TO leave_service, public")
	if err != nil {
		return nil, fmt.Errorf("[leave][database] failed to set search_path: %w", err)
	}

	// Also set search_path on the GORM session so all queries use leave_service schema
	db = db.Session(&gorm.Session{})
	db.Exec("SET search_path TO leave_service, public")

	log.Printf("[leave][database] connected to PostgreSQL, schema=leave_service")
	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	log.Printf("[leave][database] running auto-migration")

	err := db.AutoMigrate(
		&model.LeaveType{},
		&model.LeaveRequest{},
		&model.LeaveBalance{},
		&model.OvertimeRequest{},
	)
	if err != nil {
		return fmt.Errorf("[leave][database] auto-migrate failed: %w", err)
	}

	log.Printf("[leave][database] auto-migration completed successfully")
	return nil
}
