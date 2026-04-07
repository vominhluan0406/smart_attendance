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
)

type FraudAlertHandler struct {
	fraudAlertService *service.FraudAlertService
	branchService     *service.BranchService
	render            *renderer.Renderer
}

func NewFraudAlertHandler(fraudAlertService *service.FraudAlertService, branchService *service.BranchService, render *renderer.Renderer) *FraudAlertHandler {
	return &FraudAlertHandler{
		fraudAlertService: fraudAlertService,
		branchService:     branchService,
		render:            render,
	}
}

// AlertsPage renders the full fraud alerts page for manager/admin.
func (h *FraudAlertHandler) AlertsPage(w http.ResponseWriter, r *http.Request) {
	branchID := h.resolveBranchID(r)

	page, limit, alertType, severity, reviewed, dateFrom, dateTo := h.parseFilters(r)

	result, err := h.fraudAlertService.GetBranchAlerts(branchID, alertType, severity, reviewed, dateFrom, dateTo, page, limit)
	if err != nil {
		data := userContext(r)
		data["Error"] = err.Error()
		injectBranchFlags(data, r, h.branchService)
		h.render.Render(w, "fraud_alerts.html", data)
		return
	}

	data := userContext(r)
	data["Result"] = result
	data["AlertType"] = alertType
	data["Severity"] = severity
	data["Reviewed"] = r.URL.Query().Get("reviewed")
	data["DateFrom"] = r.URL.Query().Get("date_from")
	data["DateTo"] = r.URL.Query().Get("date_to")
	data["BranchID"] = branchID
	injectBranchFlags(data, r, h.branchService)
	h.render.Render(w, "fraud_alerts.html", data)
}

// AlertsPartial renders just the HTMX partial table fragment.
func (h *FraudAlertHandler) AlertsPartial(w http.ResponseWriter, r *http.Request) {
	branchID := h.resolveBranchID(r)

	page, limit, alertType, severity, reviewed, dateFrom, dateTo := h.parseFilters(r)

	result, err := h.fraudAlertService.GetBranchAlerts(branchID, alertType, severity, reviewed, dateFrom, dateTo, page, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.render.RenderPartial(w, "fraud_alerts_list.html", map[string]interface{}{
		"Result": result,
	})
}

// ReviewAction marks a fraud alert as reviewed.
func (h *FraudAlertHandler) ReviewAction(w http.ResponseWriter, r *http.Request) {
	alertID := chi.URLParam(r, "id")
	reviewerID := middleware.GetUserID(r)

	// For manager: restrict to own branch; admin: no restriction
	branchID := ""
	if middleware.GetUserRole(r) == models.RoleManager {
		branchID = middleware.GetBranchID(r)
	}

	err := h.fraudAlertService.ReviewAlert(alertID, reviewerID, branchID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("HX-Trigger", "alertReviewed")
	fmt.Fprintf(w, `<span class="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-bold bg-green-100 text-green-700"><i data-lucide="check-circle" class="w-3 h-3"></i> Đã xem xét</span><script>lucide.createIcons()</script>`)
}

// resolveBranchID returns the branch to filter by based on user role.
func (h *FraudAlertHandler) resolveBranchID(r *http.Request) string {
	role := middleware.GetUserRole(r)
	if role == models.RoleManager {
		return middleware.GetBranchID(r)
	}
	// Admin can see all — no branch filter
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
