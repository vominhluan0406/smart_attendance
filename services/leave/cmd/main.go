package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/smart-attendance/leave-service/internal/client"
	"github.com/smart-attendance/leave-service/internal/config"
	"github.com/smart-attendance/leave-service/internal/database"
	"github.com/smart-attendance/leave-service/internal/handler"
	"github.com/smart-attendance/leave-service/internal/repository"
	"github.com/smart-attendance/leave-service/internal/service"
	"github.com/smart-attendance/shared/middleware"
)

func main() {
	log.Printf("[leave] starting leave-service...")

	// Load configuration
	cfg := config.Load()

	// Connect to database
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("[leave] database connection failed: %v", err)
	}

	// Run migrations
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("[leave] auto-migration failed: %v", err)
	}

	// Seed default leave types
	if err := database.SeedLeaveTypes(db); err != nil {
		log.Fatalf("[leave] seed leave types failed: %v", err)
	}

	// Initialize HTTP clients
	authClient := client.NewAuthClient(cfg.AuthServiceURL)
	attendanceClient := client.NewAttendanceClient(cfg.AttendanceServiceURL)

	// Initialize repositories
	leaveRepo := repository.NewLeaveRepository(db)
	leaveTypeRepo := repository.NewLeaveTypeRepository(db)

	// Initialize services
	leaveService := service.NewLeaveService(leaveRepo, leaveTypeRepo, authClient, attendanceClient)

	// Initialize handlers
	leaveHandler := handler.NewLeaveHandler(leaveService)
	internalHandler := handler.NewInternalHandler(leaveService)

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
		w.Write([]byte(`{"status":"ok","service":"leave-service"}`))
	})

	// Internal routes (used by other services, no RBAC -- only service-to-service)
	r.Get("/api/internal/leave/pending-count", internalHandler.PendingCount)

	// Protected routes (require X-User-ID from gateway)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireInternalAuth)

		// Employee endpoints
		r.Get("/api/leave/my", leaveHandler.GetMyRequests)
		r.Post("/api/leave/my", leaveHandler.SubmitRequest)

		// Leave types (all authenticated users)
		r.Get("/api/leave/types", leaveHandler.GetLeaveTypes)

		// Manager endpoints
		r.Route("/api/leave/manage", func(r chi.Router) {
			r.Use(middleware.RequireRole("admin", "manager"))
			r.Get("/", leaveHandler.GetBranchRequests)
			r.Post("/{id}/review", leaveHandler.ReviewRequest)
		})
	})

	addr := ":" + cfg.Port
	log.Printf("[leave] server listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("[leave] server failed: %v", err)
	}
}
