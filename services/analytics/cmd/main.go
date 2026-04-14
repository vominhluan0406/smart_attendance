package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/smart-attendance/analytics-service/internal/client"
	"github.com/smart-attendance/analytics-service/internal/config"
	"github.com/smart-attendance/analytics-service/internal/database"
	"github.com/smart-attendance/analytics-service/internal/handler"
	"github.com/smart-attendance/analytics-service/internal/service"
	"github.com/smart-attendance/shared/middleware"
)

func main() {
	log.Printf("[analytics] starting analytics-service...")

	// Load configuration
	cfg := config.Load()

	// Connect to database (for future materialized cache tables)
	_, err := database.Connect(cfg)
	if err != nil {
		log.Printf("[analytics] WARNING: database connection failed (non-fatal for read-only service): %v", err)
	}

	// Initialize HTTP clients
	authClient := client.NewAuthClient(cfg.AuthServiceURL)
	attendanceClient := client.NewAttendanceClient(cfg.AttendanceServiceURL)
	leaveClient := client.NewLeaveClient(cfg.LeaveServiceURL)
	orgClient := client.NewOrgClient(cfg.OrgServiceURL)

	// Initialize services
	dashboardService := service.NewDashboardService(authClient, attendanceClient, leaveClient, orgClient)
	reportService := service.NewReportService(attendanceClient, authClient)

	// Initialize handlers
	dashboardHandler := handler.NewDashboardHandler(dashboardService)
	reportHandler := handler.NewReportHandler(reportService, orgClient)

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
		w.Write([]byte(`{"status":"ok","service":"analytics-service"}`))
	})

	// Protected routes (require X-User-ID from gateway)
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireInternalAuth)

		// Dashboard endpoints
		r.Get("/api/dashboard/stats", dashboardHandler.GetStats)
		r.Get("/api/dashboard/charts", dashboardHandler.GetCharts)
		r.Get("/api/dashboard/recent", dashboardHandler.GetRecent)

		// Report endpoints
		r.Get("/api/reports", reportHandler.ListBranches)

		r.Route("/api/reports/branch/{branchId}", func(r chi.Router) {
			r.Use(middleware.RequireRole("admin", "manager"))
			r.Get("/", reportHandler.GetBranchReport)
			r.Get("/export", reportHandler.ExportBranchReport)
		})

		r.Get("/api/reports/my-history", reportHandler.GetMyHistory)
		r.Get("/api/reports/my-history/export", reportHandler.ExportMyHistory)
	})

	addr := ":" + cfg.Port
	log.Printf("[analytics] server listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("[analytics] server failed: %v", err)
	}
}
