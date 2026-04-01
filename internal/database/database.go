package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"github.com/smart-attendance/smart-attendance/internal/config"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
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

	// If Turso URL is configured, use libSQL remote connection
	if cfg.TursoURL != "" {
		return connectTurso(cfg, gormCfg)
	}

	// Otherwise, use local SQLite file
	return connectLocal(cfg, gormCfg)
}

// connectLocal opens a local SQLite file (original behavior).
func connectLocal(cfg *config.Config, gormCfg *gorm.Config) (*gorm.DB, error) {
	dir := filepath.Dir(cfg.DBPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(cfg.DBPath), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("open local database: %w", err)
	}

	// SQLite performance tuning
	db.Exec("PRAGMA journal_mode=WAL")
	db.Exec("PRAGMA synchronous=NORMAL")
	db.Exec("PRAGMA cache_size=-20000")
	db.Exec("PRAGMA busy_timeout=5000")
	db.Exec("PRAGMA foreign_keys=ON")

	absPath, _ := filepath.Abs(cfg.DBPath)
	log.Printf("[database] connected to local SQLite: %s (abs: %s)", cfg.DBPath, absPath)
	return db, nil
}

// connectTurso opens a remote Turso/libSQL connection via HTTP.
// Uses libsql-client-go which works without CGO (pure Go, HTTP protocol).
// DSN format: libsql://your-db.turso.io?authToken=your-token
func connectTurso(cfg *config.Config, gormCfg *gorm.Config) (*gorm.DB, error) {
	dsn := cfg.TursoURL + "?authToken=" + cfg.TursoToken

	sqlDB, err := sql.Open("libsql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open turso connection: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping turso: %w", err)
	}

	// Use glebarez/sqlite dialector with the remote connection
	// We pass the sql.DB (Turso) directly to the dialector's Conn field
	// to ensure GORM uses it for all operations instead of a local file.
	db, err := gorm.Open(&sqlite.Dialector{
		Conn: sqlDB,
	}, &gorm.Config{
		Logger: gormCfg.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("open turso gorm: %w", err)
	}

	var sqliteVersion string
	db.Raw("SELECT sqlite_version()").Scan(&sqliteVersion)
	log.Printf("[database] connected to Turso: %s (SQLite: %s)", cfg.TursoURL, sqliteVersion)

	db.Exec("PRAGMA foreign_keys=ON")
	return db, nil
}

func AutoMigrate(db *gorm.DB, models ...interface{}) error {
	return db.AutoMigrate(models...)
}

// SafeMigrate checks if tables exist, then either creates via raw SQL (Turso) or AutoMigrate (local).
func SafeMigrate(db *gorm.DB, models ...interface{}) error {
	var count int64
	if err := db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='users'").Scan(&count).Error; err != nil {
		log.Printf("[database] cannot check tables, running full AutoMigrate")
		return db.AutoMigrate(models...)
	}

	if count > 0 {
		log.Printf("[database] tables already exist — skipping migrate")
		return nil
	}

	log.Printf("[database] tables not found — running AutoMigrate")
	return db.AutoMigrate(models...)
}

// RawMigrateTurso creates all tables using raw SQL — works reliably on Turso HTTP.
func RawMigrateTurso(db *gorm.DB) error {
	tables := []string{
		`CREATE TABLE IF NOT EXISTS branches (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, name TEXT NOT NULL, address TEXT, lat REAL, lng REAL, radius_m INTEGER DEFAULT 200, totp_secret TEXT, allowed_methods TEXT NOT NULL DEFAULT 'qr_totp,ip,location', work_start_time TEXT DEFAULT '08:00', work_end_time TEXT DEFAULT '17:00', is_active INTEGER DEFAULT 1)`,
		`CREATE TABLE IF NOT EXISTS departments (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, branch_id TEXT NOT NULL, name TEXT NOT NULL, code TEXT, manager_id TEXT, is_active INTEGER DEFAULT 1)`,
		`CREATE TABLE IF NOT EXISTS users (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, employee_code TEXT, email TEXT NOT NULL, password_hash TEXT, full_name TEXT NOT NULL, phone TEXT, role TEXT NOT NULL DEFAULT 'employee', branch_id TEXT, department_id TEXT, position TEXT, join_date DATETIME, is_active INTEGER DEFAULT 1, o_auth_provider TEXT, o_auth_id TEXT)`,
		`CREATE TABLE IF NOT EXISTS branch_ip_whitelists (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, branch_id TEXT NOT NULL, ip_c_id_r TEXT NOT NULL, label TEXT)`,
		`CREATE TABLE IF NOT EXISTS branch_locations (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, branch_id TEXT NOT NULL, label TEXT, lat REAL NOT NULL, lng REAL NOT NULL, radius_m INTEGER DEFAULT 200)`,
		`CREATE TABLE IF NOT EXISTS work_shifts (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, branch_id TEXT NOT NULL, name TEXT NOT NULL, code TEXT, start_time TEXT NOT NULL, end_time TEXT NOT NULL, grace_period_minutes INTEGER DEFAULT 15, late_threshold_minutes INTEGER DEFAULT 0, is_overnight INTEGER DEFAULT 0, break_duration_minutes INTEGER DEFAULT 60, working_days TEXT DEFAULT '1,2,3,4,5', color TEXT DEFAULT '#3B82F6', is_default INTEGER DEFAULT 0, is_active INTEGER DEFAULT 1)`,
		`CREATE TABLE IF NOT EXISTS user_shift_assignments (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, user_id TEXT NOT NULL, shift_id TEXT NOT NULL, effective_from TEXT NOT NULL, effective_to TEXT)`,
		`CREATE TABLE IF NOT EXISTS attendances (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, user_id TEXT NOT NULL, branch_id TEXT NOT NULL, shift_id TEXT, work_date TEXT NOT NULL, check_in_at DATETIME, check_out_at DATETIME, status TEXT NOT NULL DEFAULT 'on_time', method TEXT, ip_address TEXT, lat REAL, lng REAL, totp_verified INTEGER DEFAULT 0, ip_verified INTEGER DEFAULT 0, loc_verified INTEGER DEFAULT 0, note TEXT, is_adjusted INTEGER DEFAULT 0, adjusted_by_id TEXT)`,
		`CREATE TABLE IF NOT EXISTS attendance_logs (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, user_id TEXT NOT NULL, branch_id TEXT NOT NULL, shift_id TEXT, work_date TEXT NOT NULL, logged_at DATETIME NOT NULL, method TEXT, ip_address TEXT, lat REAL, lng REAL, totp_verified INTEGER DEFAULT 0, ip_verified INTEGER DEFAULT 0, loc_verified INTEGER DEFAULT 0)`,
		`CREATE TABLE IF NOT EXISTS holidays (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, name TEXT NOT NULL, date TEXT NOT NULL, branch_id TEXT, holiday_type TEXT DEFAULT 'company', is_recurring INTEGER DEFAULT 0, is_active INTEGER DEFAULT 1)`,
		`CREATE TABLE IF NOT EXISTS leave_types (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, name TEXT NOT NULL, code TEXT, max_days_per_year INTEGER DEFAULT 12, is_paid INTEGER DEFAULT 1, requires_approval INTEGER DEFAULT 1, color TEXT DEFAULT '#10B981', is_active INTEGER DEFAULT 1)`,
		`CREATE TABLE IF NOT EXISTS leave_requests (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, user_id TEXT NOT NULL, leave_type_id TEXT NOT NULL, start_date TEXT NOT NULL, end_date TEXT NOT NULL, total_days REAL NOT NULL, reason TEXT, status TEXT NOT NULL DEFAULT 'pending', reviewer_id TEXT, reviewed_at DATETIME, reviewer_note TEXT)`,
		`CREATE TABLE IF NOT EXISTS leave_balances (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, user_id TEXT NOT NULL, leave_type_id TEXT NOT NULL, year INTEGER NOT NULL, total_days REAL DEFAULT 12, used_days REAL DEFAULT 0)`,
		`CREATE TABLE IF NOT EXISTS attendance_adjustments (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, user_id TEXT NOT NULL, attendance_id TEXT, work_date TEXT NOT NULL, requested_check_in DATETIME, requested_check_out DATETIME, reason TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'pending', reviewer_id TEXT, reviewed_at DATETIME, reviewer_note TEXT)`,
		`CREATE TABLE IF NOT EXISTS overtime_requests (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, user_id TEXT NOT NULL, work_date TEXT NOT NULL, planned_start TEXT NOT NULL, planned_end TEXT NOT NULL, planned_hours REAL, reason TEXT, status TEXT NOT NULL DEFAULT 'pending', reviewer_id TEXT, reviewed_at DATETIME, reviewer_note TEXT)`,
		`CREATE TABLE IF NOT EXISTS permissions (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, code TEXT NOT NULL, name TEXT NOT NULL, description TEXT, module TEXT NOT NULL, is_active INTEGER DEFAULT 1)`,
		`CREATE TABLE IF NOT EXISTS role_permissions (id TEXT PRIMARY KEY, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, role TEXT NOT NULL, permission_id TEXT NOT NULL)`,
	}

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_users_branch_id ON users(branch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at)`,
		`CREATE INDEX IF NOT EXISTS idx_branches_deleted_at ON branches(deleted_at)`,
		`CREATE INDEX IF NOT EXISTS idx_attendances_deleted_at ON attendances(deleted_at)`,
		`CREATE INDEX IF NOT EXISTS idx_att_user_date ON attendances(user_id, work_date)`,
		`CREATE INDEX IF NOT EXISTS idx_att_branch_date ON attendances(branch_id, work_date)`,
		`CREATE INDEX IF NOT EXISTS idx_log_user_date ON attendance_logs(user_id, work_date)`,
		`CREATE INDEX IF NOT EXISTS idx_permissions_code ON permissions(code)`,
	}

	for _, sql := range tables {
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("create table: %w", err)
		}
	}
	for _, sql := range indexes {
		db.Exec(sql) // indexes are best-effort
	}

	log.Printf("[database] Turso: created %d tables + %d indexes via raw SQL", len(tables), len(indexes))
	return nil
}
