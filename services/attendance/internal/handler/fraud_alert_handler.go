package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/smart-attendance/attendance-service/internal/service"
	"github.com/smart-attendance/shared/middleware"
	"github.com/smart-attendance/shared/response"
)

type FraudAlertHandler struct {
	fraudAlertService *service.FraudAlertService
}

func NewFraudAlertHandler(fraudAlertService *service.FraudAlertService) *FraudAlertHandler {
	return &FraudAlertHandler{
		fraudAlertService: fraudAlertService,
	}
}

// ListAlerts handles GET /api/alerts
func (h *FraudAlertHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	branchID := h.resolveBranchID(r)
	page, limit, alertType, severity, reviewed, dateFrom, dateTo := h.parseFilters(r)

	result, err := h.fraudAlertService.GetBranchAlerts(branchID, alertType, severity, reviewed, dateFrom, dateTo, page, limit)
	if err != nil {
		log.Printf("[handler][fraud_alert] ListAlerts failed: %v", err)
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.JSONList(w, result.Records, result.Page, result.Limit, result.Total)
}

// ReviewAlert handles POST /api/alerts/:id/review
func (h *FraudAlertHandler) ReviewAlert(w http.ResponseWriter, r *http.Request) {
	alertID := chi.URLParam(r, "id")
	reviewerID := middleware.GetUserID(r)

	// For manager: restrict to own branch; admin: no restriction
	branchID := ""
	if middleware.GetUserRole(r) == "manager" {
		branchID = middleware.GetBranchID(r)
	}

	err := h.fraudAlertService.ReviewAlert(alertID, reviewerID, branchID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "REVIEW_FAILED", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "reviewed"})
}

// InvalidateAttendance handles POST /api/alerts/:id/invalidate
func (h *FraudAlertHandler) InvalidateAttendance(w http.ResponseWriter, r *http.Request) {
	alertID := chi.URLParam(r, "id")
	reviewerID := middleware.GetUserID(r)

	branchID := ""
	if middleware.GetUserRole(r) == "manager" {
		branchID = middleware.GetBranchID(r)
	}

	err := h.fraudAlertService.InvalidateAttendance(alertID, reviewerID, branchID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALIDATE_FAILED", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "invalidated"})
}

// resolveBranchID returns the branch to filter by based on user role.
func (h *FraudAlertHandler) resolveBranchID(r *http.Request) string {
	role := middleware.GetUserRole(r)
	if role == "manager" {
		return middleware.GetBranchID(r)
	}
	// Admin can see all -- optionally filter by query param
	if bid := r.URL.Query().Get("branch_id"); bid != "" {
		return bid
	}
	return ""
}

func (h *FraudAlertHandler) parseFilters(r *http.Request) (int, int, string, string, *bool, *time.Time, *time.Time) {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 {
		limit = 20
	}

	alertType := q.Get("alert_type")
	severity := q.Get("severity")

	var reviewed *bool
	if rv := q.Get("reviewed"); rv != "" {
		b := rv == "true"
		reviewed = &b
	}

	var dateFrom, dateTo *time.Time
	if df := q.Get("date_from"); df != "" {
		if t, err := time.Parse("2006-01-02", df); err == nil {
			dateFrom = &t
		}
	}
	if dt := q.Get("date_to"); dt != "" {
		if t, err := time.Parse("2006-01-02", dt); err == nil {
			t = t.Add(24*time.Hour - time.Second)
			dateTo = &t
		}
	}

	return page, limit, alertType, severity, reviewed, dateFrom, dateTo
}

// CountUnreviewed handles GET /api/alerts/count-unreviewed (used by dashboard/analytics).
func (h *FraudAlertHandler) CountUnreviewed(w http.ResponseWriter, r *http.Request) {
	// This would delegate to a repo method; for now included as example endpoint
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Internal JSON decode helper ---
func decodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}
