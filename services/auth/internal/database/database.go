package database

import (
	"fmt"
	"log"

	"github.com/smart-attendance/auth-service/internal/config"
	"github.com/smart-attendance/auth-service/internal/model"
	"github.com/smart-attendance/shared/migrate"
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
		return nil, fmt.Errorf("[auth][database] failed to connect: %w", err)
	}

	// Create schema if not exists and set search path
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("[auth][database] failed to get sql.DB: %w", err)
	}

	_, err = sqlDB.Exec("CREATE SCHEMA IF NOT EXISTS auth_service")
	if err != nil {
		return nil, fmt.Errorf("[auth][database] failed to create schema: %w", err)
	}

	_, err = sqlDB.Exec("SET search_path TO auth_service, public")
	if err != nil {
		return nil, fmt.Errorf("[auth][database] failed to set search_path: %w", err)
	}

	db = db.Session(&gorm.Session{})
	db.Exec("SET search_path TO auth_service, public")

	log.Printf("[auth][database] connected to PostgreSQL, schema=auth_service")
	return db, nil
}

// Migrations returns the versioned migration list for the auth service.
func Migrations() []migrate.Migration {
	return []migrate.Migration{
		{
			Version: 1,
			Name:    "create_initial_tables",
			Up: func(db *gorm.DB) error {
				return db.AutoMigrate(
					&model.User{},
					&model.UserSession{},
					&model.Permission{},
					&model.RolePermission{},
					&model.UserCredential{},
				)
			},
		},
		// Future migrations:
		// {Version: 2, Name: "add_department_id_index", Up: func(db *gorm.DB) error { ... }},
	}
}

// AutoMigrate runs versioned migrations.
func AutoMigrate(db *gorm.DB) error {
	return migrate.Run(db, Migrations())
}
