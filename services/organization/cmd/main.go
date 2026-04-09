package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/smart-attendance/organization-service/internal/config"
	"github.com/smart-attendance/organization-service/internal/database"
	"github.com/smart-attendance/organization-service/internal/handler"
	"github.com/smart-attendance/organization-service/internal/repository"
	"github.com/smart-attendance/organization-service/internal/service"
	"github.com/smart-attendance/shared/middleware"
)

func main() {
	log.Printf("[org] starting organization-service...")

	// Load configuration
	cfg := config.Load()

	// Connect to database
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("[org] database connection failed: %v", err)
	}

	// Run migrations
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("[org] auto-migration failed: %v", err)
	}

	// Seed data
	if err := database.Seed(db); err != nil {
		log.Fatalf("[org] seed failed: %v", err)
	}

	// Initialize repositories
	branchRepo := repository.NewBranchRepository(db)
	shiftRepo := repository.NewShiftRepository(db)

	// Initialize services
	branchService := service.NewBranchService(branchRepo)

	// Initialize handlers
	branchHandler := handler.NewBranchHandler(branchService)
	shiftHandler := handler.NewShiftHandler(shiftRepo)

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
		w.Write([]byte(`{"status":"ok","service":"organization-service"}`))
	})

	// Internal routes (used by other services, no RBAC)
	r.Get("/api/internal/branches/{id}", branchHandler.GetInternal)
	r.Get("/api/internal/shifts/user", shiftHandler.FindUserShift)

	// Protected routes (require X-User-ID from gateway)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireInternalAuth)

		// Branches
		r.Get("/api/branches", branchHandler.List)
		r.Post("/api/branches", branchHandler.Create)
		r.Get("/api/branches/{id}", branchHandler.GetByID)
		r.Put("/api/branches/{id}", branchHandler.Update)
		r.Delete("/api/branches/{id}", branchHandler.Delete)

		// Shifts
		r.Get("/api/shifts", shiftHandler.ListByBranch)
	})

	addr := ":" + cfg.Port
	log.Printf("[org] server listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("[org] server failed: %v", err)
	}
}
