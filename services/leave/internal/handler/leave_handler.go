package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/smart-attendance/leave-service/internal/model"
	"github.com/smart-attendance/leave-service/internal/service"
	"github.com/smart-attendance/shared/dto"
	"github.com/smart-attendance/shared/middleware"
	"github.com/smart-attendance/shared/response"
)

type LeaveHandler struct {
	leaveService *service.LeaveService
}

func NewLeaveHandler(leaveService *service.LeaveService) *LeaveHandler {
	return &LeaveHandler{
		leaveService: leaveService,
	}
}

// GetMyRequests handles GET /api/leave/my
// Returns the authenticated employee's own leave requests (paginated).
func (h *LeaveHandler) GetMyRequests(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	result, err := h.leaveService.GetMyRequests(userID, page, limit)
	if err != nil {
		log.Printf("[handler][leave] ERROR GetMyRequests user=%s: %v", userID, err)
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	records := toLeaveRequestDTOs(result.Records)
	response.JSONList(w, records, result.Page, result.Limit, result.Total)
}

// SubmitRequest handles POST /api/leave/my
// Accepts JSON body to create a new leave request.
func (h *LeaveHandler) SubmitRequest(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	branchID := middleware.GetBranchID(r)

	var req dto.CreateLeaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	if req.LeaveTypeID == "" || req.StartDate == "" || req.EndDate == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_FIELDS", "leave_type_id, start_date, end_date are required")
		return
	}

	err := h.leaveService.CreateRequest(userID, branchID, req.LeaveTypeID, req.StartDate, req.EndDate, req.Reason)
	if err != nil {
		log.Printf("[handler][leave] ERROR SubmitRequest user=%s: %v", userID, err)
		code := "CREATE_FAILED"
		status := http.StatusBadRequest
		if err == service.ErrOverlappingLeave {
			code = "OVERLAPPING_LEAVE"
		} else if err == service.ErrInvalidDateRange {
			code = "INVALID_DATE_RANGE"
		}
		response.Error(w, status, code, err.Error())
		return
	}

	log.Printf("[handler][leave] leave request created: user=%s branch=%s", userID, branchID)
	response.JSON(w, http.StatusCreated, map[string]string{"message": "leave request created"})
}

// GetBranchRequests handles GET /api/leave/manage
// Returns leave requests for the manager's branch (via X-Branch-ID header), filterable by status.
func (h *LeaveHandler) GetBranchRequests(w http.ResponseWriter, r *http.Request) {
	branchID := middleware.GetBranchID(r)
	if branchID == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_BRANCH", "X-Branch-ID header is required")
		return
	}

	status := r.URL.Query().Get("status")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	result, err := h.leaveService.GetBranchRequests(branchID, status, page, limit)
	if err != nil {
		log.Printf("[handler][leave] ERROR GetBranchRequests branch=%s: %v", branchID, err)
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	records := toLeaveRequestDTOs(result.Records)
	response.JSONList(w, records, result.Page, result.Limit, result.Total)
}

// ReviewRequest handles POST /api/leave/manage/{id}/review
// Approves or rejects a leave request.
func (h *LeaveHandler) ReviewRequest(w http.ResponseWriter, r *http.Request) {
	requestID := chi.URLParam(r, "id")
	reviewerID := middleware.GetUserID(r)

	var req dto.ReviewLeaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	leaveStatus := model.LeaveRequestStatus(req.Status)
	if leaveStatus != model.LeaveStatusApproved && leaveStatus != model.LeaveStatusRejected {
		response.Error(w, http.StatusBadRequest, "INVALID_STATUS", "status must be 'approved' or 'rejected'")
		return
	}

	err := h.leaveService.ReviewRequest(reviewerID, requestID, leaveStatus, req.Note)
	if err != nil {
		log.Printf("[handler][leave] ERROR ReviewRequest id=%s: %v", requestID, err)
		code := "REVIEW_FAILED"
		status := http.StatusInternalServerError
		if err == service.ErrLeaveNotFound {
			code = "NOT_FOUND"
			status = http.StatusNotFound
		}
		response.Error(w, status, code, err.Error())
		return
	}

	log.Printf("[handler][leave] leave request reviewed: id=%s status=%s reviewer=%s", requestID, req.Status, reviewerID)
	response.JSON(w, http.StatusOK, map[string]string{"message": "leave request updated"})
}

// GetLeaveTypes handles GET /api/leave/types
// Returns all active leave types.
func (h *LeaveHandler) GetLeaveTypes(w http.ResponseWriter, r *http.Request) {
	types, err := h.leaveService.GetLeaveTypes()
	if err != nil {
		log.Printf("[handler][leave] ERROR GetLeaveTypes: %v", err)
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	dtos := make([]dto.LeaveType, len(types))
	for i, t := range types {
		dtos[i] = dto.LeaveType{
			ID:             t.ID,
			Name:           t.Name,
			Code:           t.Code,
			MaxDaysPerYear: t.MaxDaysPerYear,
			IsPaid:         t.IsPaid,
			Color:          t.Color,
			IsActive:       t.IsActive,
		}
	}

	response.JSON(w, http.StatusOK, dtos)
}

// toLeaveRequestDTOs converts model leave requests to shared DTOs.
func toLeaveRequestDTOs(records []model.LeaveRequest) []dto.LeaveRequest {
	result := make([]dto.LeaveRequest, len(records))
	for i, r := range records {
		d := dto.LeaveRequest{
			ID:           r.ID,
			UserID:       r.UserID,
			LeaveTypeID:  r.LeaveTypeID,
			StartDate:    r.StartDate,
			EndDate:      r.EndDate,
			TotalDays:    r.TotalDays,
			Reason:       r.Reason,
			Status:       string(r.Status),
			ReviewerID:   r.ReviewerID,
			ReviewerNote: r.ReviewerNote,
			CreatedAt:    r.CreatedAt,
		}
		if r.LeaveType != nil {
			d.LeaveType = r.LeaveType.Name
		}
		result[i] = d
	}
	return result
}
