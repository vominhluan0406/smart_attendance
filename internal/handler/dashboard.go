package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/smart-attendance/smart-attendance/internal/middleware"
	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/renderer"
	"github.com/smart-attendance/smart-attendance/internal/service"
)

type DashboardHandler struct {
	dashboardService *service.DashboardService
	branchService    *service.BranchService
	render           *renderer.Renderer
}

func NewDashboardHandler(
	dashboardService *service.DashboardService,
	branchService *service.BranchService,
	render *renderer.Renderer,
) *DashboardHandler {
	return &DashboardHandler{
		dashboardService: dashboardService,
		branchService:    branchService,
		render:           render,
	}
}

// DashboardPage renders the full dashboard page.
func (h *DashboardHandler) DashboardPage(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetUserRole(r)
	userBranchID := middleware.GetBranchID(r)

	// Determine which branch to filter by
	branchID := h.resolveBranchFilter(r, role, userBranchID)

	stats, err := h.dashboardService.GetStats(branchID)
	if err != nil {
		log.Printf("[handler][dashboard] ERROR getting stats: %v", err)
		h.render.Render(w, "dashboard.html", map[string]interface{}{
			"Error":      "Không thể tải dữ liệu dashboard",
			"UserRole":   role,
			"UserBranch": userBranchID,
		})
		return
	}

	chartData, err := h.dashboardService.GetChartData(branchID, 14)
	if err != nil {
		log.Printf("[handler][dashboard] ERROR getting chart data: %v", err)
	}

	topLate, err := h.dashboardService.GetTopLate(branchID, 5)
	if err != nil {
		log.Printf("[handler][dashboard] ERROR getting top late: %v", err)
	}

	recent, err := h.dashboardService.GetRecentActivity(branchID, 10)
	if err != nil {
		log.Printf("[handler][dashboard] ERROR getting recent activity: %v", err)
	}

	// Get branches list for filter dropdown (Admin only)
	var branches []models.Branch
	if role == models.RoleAdmin {
		branches, _ = h.branchService.ListAll()
	}

	// Serialize chart data to JSON for Chart.js
	chartJSON, _ := json.Marshal(chartData)

	data := map[string]interface{}{
		"Stats":          stats,
		"ChartData":      chartData,
		"ChartJSON":      string(chartJSON),
		"TopLate":        topLate,
		"Recent":         recent,
		"Branches":       branches,
		"SelectedBranch": branchID,
		"UserRole":       role,
		"UserBranch":     userBranchID,
	}

	if err := h.render.Render(w, "dashboard.html", data); err != nil {
		log.Printf("[handler][dashboard] render error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// StatsPartial renders just the stats cards (HTMX partial).
func (h *DashboardHandler) StatsPartial(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetUserRole(r)
	userBranchID := middleware.GetBranchID(r)
	branchID := h.resolveBranchFilter(r, role, userBranchID)

	stats, err := h.dashboardService.GetStats(branchID)
	if err != nil {
		http.Error(w, "Lỗi tải thống kê", http.StatusInternalServerError)
		return
	}

	h.render.RenderPartial(w, "dashboard_stats.html", map[string]interface{}{
		"Stats":    stats,
		"UserRole": role,
	})
}

// ChartPartial renders the chart data partial (HTMX partial).
func (h *DashboardHandler) ChartPartial(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetUserRole(r)
	userBranchID := middleware.GetBranchID(r)
	branchID := h.resolveBranchFilter(r, role, userBranchID)

	days := 14
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 90 {
			days = parsed
		}
	}

	chartData, err := h.dashboardService.GetChartData(branchID, days)
	if err != nil {
		http.Error(w, "Lỗi tải biểu đồ", http.StatusInternalServerError)
		return
	}

	chartJSON, _ := json.Marshal(chartData)

	h.render.RenderPartial(w, "dashboard_chart.html", map[string]interface{}{
		"ChartData": chartData,
		"ChartJSON": string(chartJSON),
	})
}

// RecentPartial renders the recent activity partial (HTMX partial).
func (h *DashboardHandler) RecentPartial(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetUserRole(r)
	userBranchID := middleware.GetBranchID(r)
	branchID := h.resolveBranchFilter(r, role, userBranchID)

	recent, err := h.dashboardService.GetRecentActivity(branchID, 10)
	if err != nil {
		http.Error(w, "Lỗi tải hoạt động gần đây", http.StatusInternalServerError)
		return
	}

	h.render.RenderPartial(w, "dashboard_recent.html", map[string]interface{}{
		"Recent": recent,
	})
}

// APIStats returns dashboard stats as JSON.
func (h *DashboardHandler) APIStats(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetUserRole(r)
	userBranchID := middleware.GetBranchID(r)
	branchID := h.resolveBranchFilter(r, role, userBranchID)

	stats, err := h.dashboardService.GetStats(branchID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    stats,
	})
}

// APICharts returns chart data as JSON.
func (h *DashboardHandler) APICharts(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetUserRole(r)
	userBranchID := middleware.GetBranchID(r)
	branchID := h.resolveBranchFilter(r, role, userBranchID)

	days := 14
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 90 {
			days = parsed
		}
	}

	chartData, err := h.dashboardService.GetChartData(branchID, days)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    chartData,
	})
}

// resolveBranchFilter determines the branch ID to filter by based on role and query params.
func (h *DashboardHandler) resolveBranchFilter(r *http.Request, role models.Role, userBranchID string) string {
	switch role {
	case models.RoleAdmin:
		// Admin can filter by any branch via query param
		if bid := r.URL.Query().Get("branch_id"); bid != "" {
			return bid
		}
		return "" // all branches
	case models.RoleManager:
		// Manager always sees their own branch
		return userBranchID
	default:
		// Employee sees their own branch
		return userBranchID
	}
}
