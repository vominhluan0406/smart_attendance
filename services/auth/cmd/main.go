package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/smart-attendance/auth-service/internal/config"
	"github.com/smart-attendance/auth-service/internal/database"
	"github.com/smart-attendance/auth-service/internal/handler"
	"github.com/smart-attendance/auth-service/internal/repository"
	"github.com/smart-attendance/auth-service/internal/service"
	"github.com/smart-attendance/shared/event"
	"github.com/smart-attendance/shared/middleware"
)

func main() {
	log.Printf("[auth] starting auth-service...")

	// Load configuration
	cfg := config.Load()

	// Connect to database
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("[auth] database connection failed: %v", err)
	}

	// Run migrations
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("[auth] auto-migration failed: %v", err)
	}

	// Seed data
	if err := database.Seed(db); err != nil {
		log.Fatalf("[auth] seed failed: %v", err)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	sessRepo := repository.NewSessionRepository(db)
	permRepo := repository.NewPermissionRepository(db)
	credRepo := repository.NewCredentialRepository(db)

	// Connect to NATS event bus
	eventBus := event.Connect(cfg.NatsURL)
	if eventBus != nil {
		defer eventBus.Close()
	}

	// Initialize services
	authService := service.NewAuthService(cfg, userRepo, sessRepo)
	userService := service.NewUserService(userRepo, eventBus)
	permService := service.NewPermissionService(permRepo)
	webauthnService, err := service.NewWebAuthnService(cfg, userRepo, credRepo)
	if err != nil {
		log.Fatalf("[auth] failed to initialize webauthn: %v", err)
	}

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	permHandler := handler.NewPermissionHandler(permService)
	webauthnHandler := handler.NewWebAuthnHandler(webauthnService)

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
		w.Write([]byte(`{"status":"ok","service":"auth-service"}`))
	})

	// Public routes (no auth required)
	r.Post("/api/auth/login", authHandler.Login)
	r.Post("/api/auth/refresh", authHandler.Refresh)

	// Internal routes (used by Gateway and other services, no RBAC)
	r.Get("/api/internal/validate-token", authHandler.ValidateToken)
	r.Get("/api/internal/users/{id}", userHandler.GetInternal)
	r.Get("/api/internal/permissions/check", permHandler.Check)

	// Protected routes (require X-User-ID from gateway)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireInternalAuth)

		// Auth
		r.Post("/api/auth/logout", authHandler.Logout)

		// WebAuthn
		r.Get("/api/webauthn/register/begin", webauthnHandler.BeginRegistration)
		r.Post("/api/webauthn/register/finish", webauthnHandler.FinishRegistration)

		// Profile
		r.Get("/api/profile", userHandler.Profile)

		// User management (admin/manager only via gateway RBAC)
		r.Get("/api/users", userHandler.List)
		r.Post("/api/users", userHandler.Create)
		r.Get("/api/users/{id}", userHandler.GetByID)
		r.Put("/api/users/{id}", userHandler.Update)
		r.Delete("/api/users/{id}", userHandler.Delete)
	})

	addr := ":" + cfg.Port
	log.Printf("[auth] server listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("[auth] server failed: %v", err)
	}
}
