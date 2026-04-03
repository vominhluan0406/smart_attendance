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

type LeaveHandler struct {
	leaveService  *service.LeaveService
	branchService *service.BranchService
	render        *renderer.Renderer
}

func NewLeaveHandler(leaveService *service.LeaveService, branchService *service.BranchService, render *renderer.Renderer) *LeaveHandler {
	return &LeaveHandler{
		leaveService:  leaveService,
		branchService: branchService,
		render:        render,
	}
}

// MyLeavePage shows personal leave requests and a form to submit new ones.
func (h *LeaveHandler) MyLeavePage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	result, err := h.leaveService.GetMyRequests(userID, page, 10)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	leaveTypes, _ := h.leaveService.GetLeaveTypes()

	data := userContext(r)
	data["Requests"] = result.Records
	data["Total"] = result.Total
	data["Page"] = page
	data["HasNextPage"] = int64(page*10) < result.Total
	data["HasPrevPage"] = page > 1
	data["LeaveTypes"] = leaveTypes
	data["Success"] = r.URL.Query().Get("success") == "true"
	data["Error"] = r.URL.Query().Get("error")

	injectBranchFlags(data, r, h.branchService)
	h.render.Render(w, "my_leave.html", data)
}

// SubmitRequest handles leave request form submission.
func (h *LeaveHandler) SubmitRequest(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render.RenderPartial(w, "leave_error.html", "Dữ liệu không hợp lệ")
		return
	}

	userID := middleware.GetUserID(r)
	leaveTypeID := r.FormValue("leave_type_id")
	startDate := r.FormValue("start_date")
	endDate := r.FormValue("end_date")
	reason := r.FormValue("reason")

	err := h.leaveService.CreateRequest(userID, leaveTypeID, startDate, endDate, reason)
	if err != nil {
		h.render.RenderPartial(w, "leave_error.html", err.Error())
		return
	}

	// Redirect to listing with success
	w.Header().Set("HX-Redirect", "/leave/my?success=true")
	w.WriteHeader(http.StatusOK)
}

// ManageLeavePage shows pending/all requests for the branch.
func (h *LeaveHandler) ManageLeavePage(w http.ResponseWriter, r *http.Request) {
	branchID := middleware.GetBranchID(r)
	status := r.URL.Query().Get("status")
	if status == "" {
		status = string(models.LeaveStatusPending)
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	result, err := h.leaveService.GetBranchRequests(branchID, status, page, 20)
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
	h.render.Render(w, "manage_leave.html", data)
}

// ReviewAction handles Approve/Reject actions.
func (h *LeaveHandler) ReviewAction(w http.ResponseWriter, r *http.Request) {
	requestID := chi.URLParam(r, "id")
	status := r.FormValue("status") // approved or rejected
	note := r.FormValue("note")
	reviewerID := middleware.GetUserID(r)

	leaveStatus := models.LeaveRequestStatus(status)
	if leaveStatus != models.LeaveStatusApproved && leaveStatus != models.LeaveStatusRejected {
		h.render.RenderPartial(w, "leave_error.html", "Trạng thái không hợp lệ")
		return
	}

	err := h.leaveService.ReviewRequest(reviewerID, requestID, leaveStatus, note)
	if err != nil {
		h.render.RenderPartial(w, "leave_error.html", err.Error())
		return
	}

	// For HTMX, we can just return a success message or trigger a reload
	w.Header().Set("HX-Trigger", "leaveUpdated")
	fmt.Fprintf(w, "<div class='p-2 bg-green-50 text-green-700 text-sm font-bold rounded'>Cập nhật thành công</div>")
}
