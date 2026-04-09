package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/smart-attendance/gateway/internal/config"
	gw "github.com/smart-attendance/gateway/internal/middleware"
	"github.com/smart-attendance/gateway/internal/proxy"
)

func main() {
	log.Printf("[gateway] starting api-gateway...")

	cfg := config.Load()

	// Create rate limiter
	rateLimiter := gw.NewRateLimiter(cfg.RateLimitPerMin)

	// Create reverse proxies for each service
	authProxy := proxy.NewServiceProxy(cfg.AuthServiceURL)
	attendanceProxy := proxy.NewServiceProxy(cfg.AttendanceServiceURL)
	leaveProxy := proxy.NewServiceProxy(cfg.LeaveServiceURL)
	analyticsProxy := proxy.NewServiceProxy(cfg.AnalyticsServiceURL)
	orgProxy := proxy.NewServiceProxy(cfg.OrgServiceURL)

	// Setup router
	r := chi.NewRouter()

	// Global middleware chain: CORS -> RateLimit -> Logger -> Recovery
	r.Use(gw.CORS(cfg.CORSOrigin))
	r.Use(rateLimiter.Handler)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)

	// Health check — returns aggregated status of all downstream services
	r.Get("/health", healthHandler(cfg))

	// ──────────────────────────────────────────────
	// Public routes (no JWT required)
	// ──────────────────────────────────────────────
	r.Group(func(r chi.Router) {
		// Auth - login, refresh, OAuth
		r.Post("/api/auth/login", authProxy.ServeHTTP)
		r.Post("/api/auth/refresh", authProxy.ServeHTTP)
		r.Handle("/api/auth/oauth/*", authProxy)
	})

	// ──────────────────────────────────────────────
	// Protected routes (JWT middleware -> inject headers -> proxy)
	// ──────────────────────────────────────────────
	r.Group(func(r chi.Router) {
		r.Use(gw.JWTAuth(cfg.JWTSecret))

		// Auth service routes
		r.Post("/api/auth/logout", authProxy.ServeHTTP)
		r.Get("/api/profile", authProxy.ServeHTTP)
		r.Put("/api/profile", authProxy.ServeHTTP)
		r.Handle("/api/users/*", authProxy)
		r.Handle("/api/users", authProxy)
		r.Handle("/api/webauthn/*", authProxy)

		// Attendance service routes
		r.Handle("/api/attendance/*", attendanceProxy)
		r.Handle("/api/attendance", attendanceProxy)
		r.Handle("/api/alerts/*", attendanceProxy)
		r.Handle("/api/alerts", attendanceProxy)
		r.Handle("/api/adjustments/*", attendanceProxy)
		r.Handle("/api/adjustments", attendanceProxy)

		// Leave service routes
		r.Handle("/api/leave/*", leaveProxy)
		r.Handle("/api/leave", leaveProxy)

		// Analytics service routes
		r.Handle("/api/dashboard/*", analyticsProxy)
		r.Handle("/api/dashboard", analyticsProxy)
		r.Handle("/api/reports/*", analyticsProxy)
		r.Handle("/api/reports", analyticsProxy)

		// Organization service routes
		r.Handle("/api/branches/*", orgProxy)
		r.Handle("/api/branches", orgProxy)
		r.Handle("/api/shifts/*", orgProxy)
		r.Handle("/api/shifts", orgProxy)
		r.Handle("/api/departments/*", orgProxy)
		r.Handle("/api/departments", orgProxy)
		r.Handle("/api/holidays/*", orgProxy)
		r.Handle("/api/holidays", orgProxy)
	})

	addr := ":" + cfg.Port
	log.Printf("[gateway] server listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("[gateway] server failed: %v", err)
	}
}

// serviceHealth holds the health status of a downstream service.
type serviceHealth struct {
	Service string `json:"service"`
	Status  string `json:"status"`
	URL     string `json:"url"`
}

// healthHandler checks all downstream services and returns aggregated status.
func healthHandler(cfg *config.Config) http.HandlerFunc {
	type healthResponse struct {
		Status   string          `json:"status"`
		Service  string          `json:"service"`
		Upstream []serviceHealth `json:"upstream"`
	}

	services := []struct {
		name string
		url  string
	}{
		{"auth", cfg.AuthServiceURL},
		{"attendance", cfg.AttendanceServiceURL},
		{"leave", cfg.LeaveServiceURL},
		{"analytics", cfg.AnalyticsServiceURL},
		{"organization", cfg.OrgServiceURL},
	}

	client := &http.Client{Timeout: 3 * time.Second}

	return func(w http.ResponseWriter, r *http.Request) {
		upstream := make([]serviceHealth, 0, len(services))
		allHealthy := true

		for _, svc := range services {
			health := serviceHealth{
				Service: svc.name,
				URL:     svc.url,
				Status:  "healthy",
			}

			resp, err := client.Get(svc.url + "/health")
			if err != nil {
				health.Status = fmt.Sprintf("unhealthy: %v", err)
				allHealthy = false
			} else {
				resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					health.Status = fmt.Sprintf("unhealthy: status %d", resp.StatusCode)
					allHealthy = false
				}
			}

			upstream = append(upstream, health)
		}

		status := "healthy"
		httpStatus := http.StatusOK
		if !allHealthy {
			status = "degraded"
			httpStatus = http.StatusOK // still return 200, but indicate degraded in body
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(httpStatus)
		json.NewEncoder(w).Encode(healthResponse{
			Status:   status,
			Service:  "api-gateway",
			Upstream: upstream,
		})
	}
}
