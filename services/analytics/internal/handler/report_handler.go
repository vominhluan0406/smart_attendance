package handler

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/smart-attendance/analytics-service/internal/client"
	"github.com/smart-attendance/analytics-service/internal/service"
	"github.com/smart-attendance/shared/middleware"
	"github.com/smart-attendance/shared/response"
)

type ReportHandler struct {
	reportService *service.ReportService
	orgClient     *client.OrgClient
}

func NewReportHandler(reportService *service.ReportService, orgClient *client.OrgClient) *ReportHandler {
	return &ReportHandler{
		reportService: reportService,
		orgClient:     orgClient,
	}
}

// ListBranches handles GET /api/reports
// Returns list of branches for admin selection.
func (h *ReportHandler) ListBranches(w http.ResponseWriter, r *http.Request) {
	branches, err := h.orgClient.ListBranches()
	if err != nil {
		log.Printf("[handler][report] ERROR ListBranches: %v", err)
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load branches")
		return
	}

	response.JSON(w, http.StatusOK, branches)
}

// GetBranchReport handles GET /api/reports/branch/{branchId}
// Returns paginated attendance list for a branch.
func (h *ReportHandler) GetBranchReport(w http.ResponseWriter, r *http.Request) {
	branchID := chi.URLParam(r, "branchId")

	// RBAC: Manager can only see their own branch
	role := middleware.GetUserRole(r)
	if role == "manager" {
		userBranchID := middleware.GetBranchID(r)
		if userBranchID != branchID {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "you can only view reports for your own branch")
			return
		}
	}

	page, limit, dateFrom, dateTo, status := parseFilters(r)

	result, err := h.reportService.GetBranchReport(branchID, page, limit, dateFrom, dateTo, status)
	if err != nil {
		log.Printf("[handler][report] ERROR GetBranchReport branch=%s: %v", branchID, err)
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load branch report")
		return
	}

	response.JSONList(w, result.Records, result.Page, result.Limit, result.Total)
}

// ExportBranchReport handles GET /api/reports/branch/{branchId}/export
// Generates and downloads an Excel file for the branch.
func (h *ReportHandler) ExportBranchReport(w http.ResponseWriter, r *http.Request) {
	branchID := chi.URLParam(r, "branchId")

	// RBAC: Manager can only export their own branch
	role := middleware.GetUserRole(r)
	if role == "manager" {
		userBranchID := middleware.GetBranchID(r)
		if userBranchID != branchID {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "you can only export reports for your own branch")
			return
		}
	}

	_, _, dateFrom, dateTo, status := parseFilters(r)

	buf, err := h.reportService.ExportBranchExcel(branchID, dateFrom, dateTo, status)
	if err != nil {
		log.Printf("[handler][report] ERROR ExportBranchReport branch=%s: %v", branchID, err)
		response.Error(w, http.StatusInternalServerError, "EXPORT_FAILED", "failed to generate Excel")
		return
	}

	filename := fmt.Sprintf("branch_report_%s_%s.xlsx", branchID, time.Now().Format("20060102"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Write(buf)
}

// GetMyHistory handles GET /api/reports/my-history
// Returns personal attendance history for the authenticated user.
func (h *ReportHandler) GetMyHistory(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "user ID not found")
		return
	}

	page, limit, dateFrom, dateTo, status := parseFilters(r)

	result, err := h.reportService.GetUserHistory(userID, page, limit, dateFrom, dateTo, status)
	if err != nil {
		log.Printf("[handler][report] ERROR GetMyHistory user=%s: %v", userID, err)
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load history")
		return
	}

	response.JSONList(w, result.Records, result.Page, result.Limit, result.Total)
}

// ExportMyHistory handles GET /api/reports/my-history/export
// Generates and downloads an Excel file for the authenticated user.
func (h *ReportHandler) ExportMyHistory(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "user ID not found")
		return
	}

	_, _, dateFrom, dateTo, status := parseFilters(r)

	buf, err := h.reportService.ExportUserExcel(userID, dateFrom, dateTo, status)
	if err != nil {
		log.Printf("[handler][report] ERROR ExportMyHistory user=%s: %v", userID, err)
		response.Error(w, http.StatusInternalServerError, "EXPORT_FAILED", "failed to generate Excel")
		return
	}

	filename := fmt.Sprintf("my_history_%s.xlsx", time.Now().Format("20060102"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Write(buf)
}

// parseFilters extracts pagination and filter parameters from query string.
func parseFilters(r *http.Request) (int, int, string, string, string) {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 {
		limit = 20
	}

	dateFrom := q.Get("date_from")
	dateTo := q.Get("date_to")
	status := q.Get("status")

	return page, limit, dateFrom, dateTo, status
}
