package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	gocache "github.com/patrickmn/go-cache"
	"github.com/smart-attendance/attendance-service/internal/client"
	"github.com/smart-attendance/attendance-service/internal/config"
	"github.com/smart-attendance/attendance-service/internal/database"
	"github.com/smart-attendance/attendance-service/internal/handler"
	"github.com/smart-attendance/attendance-service/internal/repository"
	"github.com/smart-attendance/attendance-service/internal/service"
	"github.com/smart-attendance/attendance-service/internal/wal"
	"github.com/smart-attendance/shared/event"
	"github.com/smart-attendance/shared/middleware"
)

func main() {
	log.Printf("[attendance] starting attendance-service...")

	// Load configuration
	cfg := config.Load()

	// Connect to database
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("[attendance] database connection failed: %v", err)
	}

	// Run migrations
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("[attendance] auto-migration failed: %v", err)
	}

	// Initialize in-memory cache (for TOTP nonces, impossible travel, anomaly stats)
	cache := gocache.New(5*time.Minute, 10*time.Minute)

	// Initialize HTTP clients (with local cache)
	authClient := client.NewAuthClient(cfg.AuthServiceURL)
	orgClient := client.NewOrgClient(cfg.OrgServiceURL)

	// Connect to NATS for cache invalidation events
	eventBus := event.Connect(cfg.NatsURL)
	if eventBus != nil {
		defer eventBus.Close()

		// Subscribe: invalidate user cache when Auth Service updates a user
		event.SubscribeJSON(eventBus, event.SubjectUserUpdated, func(ev event.UserEvent) {
			authClient.InvalidateUser(ev.UserID)
		})
		event.SubscribeJSON(eventBus, event.SubjectUserDeleted, func(ev event.UserEvent) {
			authClient.InvalidateUser(ev.UserID)
		})

		// Subscribe: invalidate branch cache when Org Service updates a branch
		event.SubscribeJSON(eventBus, event.SubjectBranchUpdated, func(ev event.BranchEvent) {
			orgClient.InvalidateBranch(ev.BranchID)
		})
		event.SubscribeJSON(eventBus, event.SubjectBranchDeleted, func(ev event.BranchEvent) {
			orgClient.InvalidateBranch(ev.BranchID)
		})
	}

	// Initialize repositories
	attendanceRepo := repository.NewAttendanceRepository(db)
	logRepo := repository.NewAttendanceLogRepository(db)
	alertRepo := repository.NewFraudAlertRepository(db)
	deviceRepo := repository.NewUserDeviceRepository(db)
	adjRepo := repository.NewAttendanceAdjustmentRepository(db)

	// Initialize WAL (write-ahead log for DB failure resilience)
	walWriter, err := wal.NewWriter("data/wal")
	if err != nil {
		log.Printf("[attendance] WARNING: WAL init failed: %v (continuing without WAL)", err)
	}

	// Initialize services
	totpService := service.NewTOTPService()
	ipValidator := service.NewIPValidator()
	locValidator := service.NewLocationValidator()

	antiFraudService := service.NewAntiFraudService(cache, logRepo, deviceRepo, alertRepo)

	attendanceService := service.NewAttendanceService(
		attendanceRepo, logRepo,
		authClient, orgClient,
		totpService, ipValidator, locValidator,
		antiFraudService,
		walWriter,
	)

	// Start WAL processor cron job (retry pending entries every 30 seconds)
	if walWriter != nil {
		walProcessor := wal.NewProcessor(walWriter, attendanceRepo, logRepo, 30*time.Second)
		walProcessor.Start()
		defer walProcessor.Stop()
	}

	fraudAlertService := service.NewFraudAlertService(alertRepo, attendanceRepo)
	adjService := service.NewAttendanceAdjustmentService(adjRepo, attendanceRepo, authClient)

	// Initialize handlers
	attendanceHandler := handler.NewAttendanceHandler(attendanceService, totpService, orgClient)
	fraudAlertHandler := handler.NewFraudAlertHandler(fraudAlertService)
	adjustmentHandler := handler.NewAdjustmentHandler(adjService)
	deviceHandler := handler.NewDeviceHandler(deviceRepo)
	internalHandler := handler.NewInternalHandler(attendanceService)

	// Setup router
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"attendance-service"}`))
	})

	// Internal routes (used by other services, no RBAC -- only service-to-service)
	r.Get("/api/internal/attendance", internalHandler.ListAttendance)
	r.Post("/api/internal/attendance/sync-leave", internalHandler.SyncLeave)

	// Protected routes (require X-User-ID from gateway)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireInternalAuth)

		// Attendance
		r.Post("/api/attendance/log", attendanceHandler.APILogTime)
		r.Post("/api/attendance/check-proximity", attendanceHandler.APICheckProximity)
		r.Get("/api/attendance/status", attendanceHandler.APIStatus)
		r.Get("/api/attendance", attendanceHandler.APIList)

		// QR Code generation (manager/admin)
		r.Get("/api/attendance/qr/{branchId}/code", attendanceHandler.QRCode)
		r.Get("/api/attendance/qr/{branchId}/image", attendanceHandler.QRImage)

		// Fraud Alerts (manager/admin)
		r.Route("/api/alerts", func(r chi.Router) {
			r.Use(middleware.RequireRole("admin", "manager"))
			r.Get("/", fraudAlertHandler.ListAlerts)
			r.Post("/{id}/review", fraudAlertHandler.ReviewAlert)
			r.Post("/{id}/invalidate", fraudAlertHandler.InvalidateAttendance)
		})

		// Adjustments - Employee
		r.Get("/api/adjustments/my", adjustmentHandler.GetMyRequests)
		r.Post("/api/adjustments/my", adjustmentHandler.CreateRequest)

		// Adjustments - Manager
		r.Route("/api/adjustments/manage", func(r chi.Router) {
			r.Use(middleware.RequireRole("admin", "manager"))
			r.Get("/", adjustmentHandler.GetBranchRequests)
			r.Post("/{id}/review", adjustmentHandler.ReviewRequest)
		})

		// Device management (admin — manage user's devices)
		r.Route("/api/users/{id}/devices", func(r chi.Router) {
			r.Use(middleware.RequireRole("admin"))
			r.Get("/", deviceHandler.ListDevices)
			r.Post("/{deviceId}/block", deviceHandler.BlockDevice)
			r.Post("/{deviceId}/unblock", deviceHandler.UnblockDevice)
			r.Delete("/{deviceId}", deviceHandler.DeleteDevice)
		})
	})

	addr := ":" + cfg.Port
	log.Printf("[attendance] server listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("[attendance] server failed: %v", err)
	}
}
