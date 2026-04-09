package database

import (
	"fmt"
	"log"

	"github.com/smart-attendance/analytics-service/internal/config"
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
		return nil, fmt.Errorf("[analytics][database] failed to connect: %w", err)
	}

	// Create schema if not exists and set search path
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("[analytics][database] failed to get sql.DB: %w", err)
	}

	_, err = sqlDB.Exec("CREATE SCHEMA IF NOT EXISTS analytics_service")
	if err != nil {
		return nil, fmt.Errorf("[analytics][database] failed to create schema: %w", err)
	}

	_, err = sqlDB.Exec("SET search_path TO analytics_service, public")
	if err != nil {
		return nil, fmt.Errorf("[analytics][database] failed to set search_path: %w", err)
	}

	db = db.Session(&gorm.Session{})
	db.Exec("SET search_path TO analytics_service, public")

	log.Printf("[analytics][database] connected to PostgreSQL, schema=analytics_service")
	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	log.Printf("[analytics][database] running auto-migration (no local tables needed)")
	// Analytics service is read-only; no local models to migrate.
	// Add materialized cache tables here if needed in the future.
	return nil
}
