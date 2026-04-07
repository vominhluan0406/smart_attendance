package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/smart-attendance/smart-attendance/internal/middleware"
	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/renderer"
	"github.com/smart-attendance/smart-attendance/internal/service"
)

type AttendanceAdjustmentHandler struct {
	adjService    *service.AttendanceAdjustmentService
	branchService *service.BranchService
	render        *renderer.Renderer
}

func NewAttendanceAdjustmentHandler(adjService *service.AttendanceAdjustmentService, branchService *service.BranchService, render *renderer.Renderer) *AttendanceAdjustmentHandler {
	return &AttendanceAdjustmentHandler{
		adjService:    adjService,
		branchService: branchService,
		render:        render,
	}
}

// MyAdjustmentsPage renders the employee's adjustment request page.
func (h *AttendanceAdjustmentHandler) MyAdjustmentsPage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	result, err := h.adjService.GetMyRequests(userID, page, 10)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := userContext(r)
	data["Requests"] = result.Records
	data["Total"] = result.Total
	data["Page"] = page
	data["HasNextPage"] = int64(page*10) < result.Total
	data["HasPrevPage"] = page > 1
	data["Success"] = r.URL.Query().Get("success") == "true"
	data["Error"] = r.URL.Query().Get("error")

	injectBranchFlags(data, r, h.branchService)
	h.render.Render(w, "my_adjustments.html", data)
}

// SubmitRequest handles form submission from employee.
func (h *AttendanceAdjustmentHandler) SubmitRequest(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render.RenderPartial(w, "adjustment_error.html", "Dữ liệu không hợp lệ")
		return
	}

	userID := middleware.GetUserID(r)
	workDate := r.FormValue("work_date")
	checkIn := r.FormValue("check_in")
	checkOut := r.FormValue("check_out")
	reason := r.FormValue("reason")

	err := h.adjService.CreateRequest(userID, workDate, checkIn, checkOut, reason)
	if err != nil {
		h.render.RenderPartial(w, "adjustment_error.html", err.Error())
		return
	}

	w.Header().Set("HX-Redirect", "/adjustments/my?success=true")
	w.WriteHeader(http.StatusOK)
}

// ManageAdjustmentsPage renders the manager's approval page.
func (h *AttendanceAdjustmentHandler) ManageAdjustmentsPage(w http.ResponseWriter, r *http.Request) {
	branchID := middleware.GetBranchID(r)
	status := r.URL.Query().Get("status")
	if status == "" {
		status = string(models.AdjustStatusPending)
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	result, err := h.adjService.GetBranchRequests(branchID, status, page, 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := userContext(r)
	data["Requests"] = result.Records
	data["Total"] = result.Total
	data["Page"] = page
	data["StatusFilter"] = status
	data["HasNextPage"] = int64(page*20) < result.Total
	data["HasPrevPage"] = page > 1

	injectBranchFlags(data, r, h.branchService)
	h.render.Render(w, "manage_adjustments.html", data)
}

// ReviewAction handles approve/reject from manager.
func (h *AttendanceAdjustmentHandler) ReviewAction(w http.ResponseWriter, r *http.Request) {
	requestID := chi.URLParam(r, "id")
	status := r.FormValue("status")
	note := r.FormValue("note")
	reviewerID := middleware.GetUserID(r)

	adjStatus := models.AdjustmentStatus(status)
	if adjStatus != models.AdjustStatusApproved && adjStatus != models.AdjustStatusRejected {
		h.render.RenderPartial(w, "adjustment_error.html", "Trạng thái không hợp lệ")
		return
	}

	err := h.adjService.ReviewRequest(reviewerID, requestID, adjStatus, note)
	if err != nil {
		h.render.RenderPartial(w, "adjustment_error.html", err.Error())
		return
	}

	w.Header().Set("HX-Trigger", "adjustmentUpdated")
	fmt.Fprintf(w, "<div class='p-2 bg-green-50 text-green-700 text-sm font-bold rounded'>Cập nhật thành công</div>")
}
