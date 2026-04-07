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

	// Migrate: Turso uses raw SQL, local uses GORM AutoMigrate
	if cfg.TursoURL != "" {
		log.Printf("[main] using Turso migration path")
		if err := database.RawMigrateTurso(db); err != nil {
			log.Fatalf("turso migrate failed: %v", err)
		}
	} else {
		log.Printf("[main] using local SQLite migration path")
		if err := database.SafeMigrate(db,
			&models.User{},
			&models.Branch{},
			&models.BranchIPWhitelist{},
			&models.BranchLocation{},
			&models.Attendance{},
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
			&models.UserCredential{},
			&models.UserDevice{},
			&models.UserSession{},
			&models.FraudAlert{},
		); err != nil {
			log.Fatalf("auto-migrate failed: %v", err)
		}

		if err := database.RunMigrations(db); err != nil {
			log.Fatalf("data migration failed: %v", err)
		}
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
	credRepo := repository.NewUserCredentialRepository(db)
	leaveRepo := repository.NewLeaveRepository(db)
	leaveTypeRepo := repository.NewLeaveTypeRepository(db)
	deviceRepo := repository.NewUserDeviceRepository(db)
	sessionRepo := repository.NewUserSessionRepository(db)
	fraudAlertRepo := repository.NewFraudAlertRepository(db)

	// Init services
	authService := service.NewAuthService(userRepo, branchRepo, sessionRepo, appCache, cfg)
	userService := service.NewUserService(userRepo)
	branchService := service.NewBranchService(branchRepo, userRepo, appCache)
	totpService := service.NewTOTPService()
	ipValidator := service.NewIPValidator()
	locValidator := service.NewLocationValidator()
	permissionService := service.NewPermissionService(permRepo, appCache)
	antiFraudService := service.NewAntiFraudService(appCache, attendanceLogRepo, deviceRepo, fraudAlertRepo)

	// WebAuthn RPID and Origin from config
	webAuthnService, err := service.NewWebAuthnService(cfg.WebAuthnRPID, cfg.WebAuthnOrigin, credRepo, userRepo, appCache)
	if err != nil {
		log.Fatalf("Failed to initialize WebAuthnService: %v", err)
	}

	fraudAlertService := service.NewFraudAlertService(fraudAlertRepo, attendanceRepo)
	adjRepo := repository.NewAttendanceAdjustmentRepository(db)
	adjService := service.NewAttendanceAdjustmentService(adjRepo, attendanceRepo, userRepo)
	leaveService := service.NewLeaveService(leaveRepo, leaveTypeRepo, attendanceRepo, userRepo)
	attendanceService := service.NewAttendanceService(attendanceRepo, attendanceLogRepo, shiftRepo, branchService, userService, totpService, ipValidator, locValidator, leaveRepo, antiFraudService)
	reportService := service.NewReportService(attendanceRepo)
	dashboardService := service.NewDashboardService(attendanceRepo, branchRepo, userRepo, leaveRepo, appCache, db)

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
		WebAuthnService:   webAuthnService,
		LeaveService:      leaveService,
		FraudAlertService:         fraudAlertService,
		AttendanceAdjustmentService: adjService,
		Config:                    cfg,
		RateLimitPerMin:     cfg.RateLimitPerMin,
		UserRateLimitPerMin: cfg.UserRateLimitPerMin,
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
