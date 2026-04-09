package handler

import (
	"log"
	"net/http"
	"strconv"

	"github.com/smart-attendance/analytics-service/internal/service"
	"github.com/smart-attendance/shared/middleware"
	"github.com/smart-attendance/shared/response"
)

type DashboardHandler struct {
	dashboardService *service.DashboardService
}

func NewDashboardHandler(dashboardService *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{
		dashboardService: dashboardService,
	}
}

// GetStats handles GET /api/dashboard/stats?branch_id=X
// Returns aggregated dashboard statistics as JSON.
func (h *DashboardHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	branchID := resolveBranchFilter(r)

	stats, err := h.dashboardService.GetStats(branchID)
	if err != nil {
		log.Printf("[handler][dashboard] ERROR GetStats branch=%s: %v", branchID, err)
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load dashboard stats")
		return
	}

	response.JSON(w, http.StatusOK, stats)
}

// GetCharts handles GET /api/dashboard/charts?branch_id=X&days=14
// Returns daily attendance data for chart display.
func (h *DashboardHandler) GetCharts(w http.ResponseWriter, r *http.Request) {
	branchID := resolveBranchFilter(r)

	days := 14
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 90 {
			days = parsed
		}
	}

	chartData, err := h.dashboardService.GetChartData(branchID, days)
	if err != nil {
		log.Printf("[handler][dashboard] ERROR GetCharts branch=%s: %v", branchID, err)
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load chart data")
		return
	}

	response.JSON(w, http.StatusOK, chartData)
}

// GetRecent handles GET /api/dashboard/recent?branch_id=X&limit=10
// Returns recent check-in records.
func (h *DashboardHandler) GetRecent(w http.ResponseWriter, r *http.Request) {
	branchID := resolveBranchFilter(r)

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	recent, err := h.dashboardService.GetRecentActivity(branchID, limit)
	if err != nil {
		log.Printf("[handler][dashboard] ERROR GetRecent branch=%s: %v", branchID, err)
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load recent activity")
		return
	}

	response.JSON(w, http.StatusOK, recent)
}

// resolveBranchFilter determines branch filter from role and query params.
// Admin can filter by any branch via query param; Manager/Employee see their own branch.
func resolveBranchFilter(r *http.Request) string {
	role := middleware.GetUserRole(r)

	switch role {
	case "admin":
		// Admin can filter by any branch via query param
		if bid := r.URL.Query().Get("branch_id"); bid != "" {
			return bid
		}
		return "" // all branches
	default:
		// Manager/Employee always see their own branch
		branchID := middleware.GetBranchID(r)
		if branchID != "" {
			return branchID
		}
		// Fallback to query param if header not set
		return r.URL.Query().Get("branch_id")
	}
}
