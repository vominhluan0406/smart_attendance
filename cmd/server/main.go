package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/smart-attendance/smart-attendance/internal/cache"
	"github.com/smart-attendance/smart-attendance/internal/config"
	"github.com/smart-attendance/smart-attendance/internal/database"
	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/renderer"
	"github.com/smart-attendance/smart-attendance/internal/repository"
	"github.com/smart-attendance/smart-attendance/internal/router"
	"github.com/smart-attendance/smart-attendance/internal/service"
)

func main() {
	// Auto-detect project root: find go.mod and chdir to its directory
	if err := chdirToProjectRoot(); err != nil {
		log.Printf("[main] warning: could not detect project root: %v", err)
	}

	// Load config
	cfg := config.Load()

	// Connect database
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}

	// Auto-migrate models
	if err := database.AutoMigrate(db,
		&models.User{},
		&models.Branch{},
		&models.BranchIPWhitelist{},
		&models.BranchLocation{},
		&models.Attendance{},
		// New tables
		&models.Department{},
		&models.WorkShift{},
		&models.UserShiftAssignment{},
		&models.Holiday{},
		&models.LeaveType{},
		&models.LeaveRequest{},
		&models.LeaveBalance{},
		&models.AttendanceAdjustment{},
		&models.OvertimeRequest{},
		&models.Permission{},
		&models.RolePermission{},
		&models.AttendanceLog{},
	); err != nil {
		log.Fatalf("auto-migrate failed: %v", err)
	}

	// Run data migrations (backfill work_date, create default shifts)
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("data migration failed: %v", err)
	}

	// Seed default data
	if err := database.Seed(db); err != nil {
		log.Fatalf("seed failed: %v", err)
	}

	// Init cache
	appCache := cache.New(5*time.Minute, 10*time.Minute)

	// Init template renderer
	render, err := renderer.New("web/templates", cfg.Env == "development")
	if err != nil {
		log.Fatalf("template renderer init failed: %v", err)
	}

	// Init repositories
	userRepo := repository.NewUserRepository(db)
	branchRepo := repository.NewBranchRepository(db)
	attendanceRepo := repository.NewAttendanceRepository(db)
	attendanceLogRepo := repository.NewAttendanceLogRepository(db)
	shiftRepo := repository.NewShiftRepository(db)
	permRepo := repository.NewPermissionRepository(db)

	// Init services
	authService := service.NewAuthService(userRepo, cfg)
	userService := service.NewUserService(userRepo)
	branchService := service.NewBranchService(branchRepo, userRepo, appCache)
	totpService := service.NewTOTPService()
	ipValidator := service.NewIPValidator()
	locValidator := service.NewLocationValidator()
	attendanceService := service.NewAttendanceService(attendanceRepo, attendanceLogRepo, shiftRepo, branchService, userService, totpService, ipValidator, locValidator)
	reportService := service.NewReportService(attendanceRepo)
	dashboardService := service.NewDashboardService(attendanceRepo, branchRepo, userRepo, appCache, db)
	permissionService := service.NewPermissionService(permRepo, appCache)

	// Setup router
	handler := router.New(router.Deps{
		Render:            render,
		AuthService:       authService,
		UserService:       userService,
		BranchService:     branchService,
		AttendanceService: attendanceService,
		TOTPService:       totpService,
		ReportService:     reportService,
		DashboardService:  dashboardService,
		PermissionService: permissionService,
		Config:            cfg,
		RateLimitPerMin:   cfg.RateLimitPerMin,
	})

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Smart Attendance server starting on http://localhost%s (env=%s)", addr, cfg.Env)

	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

// chdirToProjectRoot walks up from cwd looking for go.mod to find project root.
func chdirToProjectRoot() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return os.Chdir(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return fmt.Errorf("go.mod not found in any parent directory")
		}
		dir = parent
	}
}
