package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/smart-attendance/smart-attendance/internal/middleware"
	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/renderer"
	"github.com/smart-attendance/smart-attendance/internal/service"
	"github.com/smart-attendance/smart-attendance/internal/timezone"
)

type ReportHandler struct {
	reportService *service.ReportService
	branchService *service.BranchService
	render        *renderer.Renderer
}

func NewReportHandler(
	reportService *service.ReportService,
	branchService *service.BranchService,
	render *renderer.Renderer,
) *ReportHandler {
	return &ReportHandler{
		reportService: reportService,
		branchService: branchService,
		render:        render,
	}
}

// UserHistoryPage renders the full page for a user's attendance history
func (h *ReportHandler) UserHistoryPage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	page, limit, dateFrom, dateTo, status := h.parseFilters(r)

	// RBAC: Admin and Manager don't have personal history (they only see reports)
	role := middleware.GetUserRole(r)
	if role == models.RoleAdmin || role == models.RoleManager {
		data := userContext(r)
		data["Error"] = "Forbidden: Admin and Manager roles do not have personal history records."
		h.render.Render(w, "my_history.html", data)
		return
	}

	result, err := h.reportService.GetUserHistory(userID, page, limit, dateFrom, dateTo, status)
	if err != nil {
		data := userContext(r)
		data["Error"] = err.Error()
		h.render.Render(w, "my_history.html", data)
		return
	}

	data := userContext(r)
	data["Result"] = result
	data["DateFrom"] = r.URL.Query().Get("date_from")
	data["DateTo"] = r.URL.Query().Get("date_to")
	data["Status"] = status
	h.render.Render(w, "my_history.html", data)
}

// UserHistoryPartial renders just the HTMX partial table
func (h *ReportHandler) UserHistoryPartial(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	page, limit, dateFrom, dateTo, status := h.parseFilters(r)

	// RBAC: Admin and Manager check
	role := middleware.GetUserRole(r)
	if role == models.RoleAdmin || role == models.RoleManager {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	result, err := h.reportService.GetUserHistory(userID, page, limit, dateFrom, dateTo, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.render.RenderPartial(w, "history_list.html", map[string]interface{}{
		"Result": result,
	})
}

// BranchReportPage renders the full page for branch attendance reports (Manager/Admin)
func (h *ReportHandler) BranchReportPage(w http.ResponseWriter, r *http.Request) {
	branchID := chi.URLParam(r, "branchID")
	
	// RBAC: Manager can only see their own branch
	role := middleware.GetUserRole(r)
	if role == models.RoleManager {
		userBranchID := middleware.GetBranchID(r)
		if userBranchID != branchID {
			http.Error(w, "Forbidden: You can only view reports for your own branch.", http.StatusForbidden)
			return
		}
	}

	branch, err := h.branchService.GetByIDCached(branchID)
	if err != nil {
		http.Error(w, "Branch not found", http.StatusNotFound)
		return
	}

	page, limit, dateFrom, dateTo, status := h.parseFilters(r)

	result, err := h.reportService.GetBranchReport(branchID, page, limit, dateFrom, dateTo, status)
	if err != nil {
		data := userContext(r)
		data["Error"] = err.Error()
		data["Branch"] = branch
		h.render.Render(w, "branch_report.html", data)
		return
	}

	data := userContext(r)
	data["Branch"] = branch
	data["Result"] = result
	data["DateFrom"] = r.URL.Query().Get("date_from")
	data["DateTo"] = r.URL.Query().Get("date_to")
	data["Status"] = status
	h.render.Render(w, "branch_report.html", data)
}

// BranchReportPartial renders just the HTMX partial table
func (h *ReportHandler) BranchReportPartial(w http.ResponseWriter, r *http.Request) {
	branchID := chi.URLParam(r, "branchID")

	// RBAC: Manager check
	if middleware.GetUserRole(r) == models.RoleManager {
		if middleware.GetBranchID(r) != branchID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	page, limit, dateFrom, dateTo, status := h.parseFilters(r)

	result, err := h.reportService.GetBranchReport(branchID, page, limit, dateFrom, dateTo, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.render.RenderPartial(w, "history_list.html", map[string]interface{}{
		"Result": result,
	})
}

// ExportUserHistory generates and downloads Excel file
func (h *ReportHandler) ExportUserHistory(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	_, _, dateFrom, dateTo, status := h.parseFilters(r)

	buf, err := h.reportService.ExportUserHistoryExcel(userID, dateFrom, dateTo, status)
	if err != nil {
		http.Error(w, "Failed to generate Excel", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("my_history_%s.xlsx", timezone.Now().Format("20060102"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Write(buf)
}

// ExportBranchReport generates and downloads Excel file
func (h *ReportHandler) ExportBranchReport(w http.ResponseWriter, r *http.Request) {
	branchID := chi.URLParam(r, "branchID")

	// RBAC: Manager check
	if middleware.GetUserRole(r) == models.RoleManager {
		if middleware.GetBranchID(r) != branchID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	_, _, dateFrom, dateTo, status := h.parseFilters(r)

	buf, err := h.reportService.ExportBranchReportExcel(branchID, dateFrom, dateTo, status)
	if err != nil {
		http.Error(w, "Failed to generate Excel", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("branch_report_%s.xlsx", timezone.Now().Format("20060102"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Write(buf)
}

func (h *ReportHandler) parseFilters(r *http.Request) (int, int, *time.Time, *time.Time, string) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 {
		limit = 20
	}

	var dateFrom, dateTo *time.Time
	if df := q.Get("date_from"); df != "" {
		if t, err := time.Parse("2006-01-02", df); err == nil {
			dateFrom = &t
		}
	}
	if dt := q.Get("date_to"); dt != "" {
		if t, err := time.Parse("2006-01-02", dt); err == nil {
			// Include the whole day
			t = t.Add(24*time.Hour - time.Second)
			dateTo = &t
		}
	}

	return page, limit, dateFrom, dateTo, q.Get("status")
}
