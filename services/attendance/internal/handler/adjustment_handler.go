package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/smart-attendance/attendance-service/internal/model"
	"github.com/smart-attendance/attendance-service/internal/service"
	"github.com/smart-attendance/shared/dto"
	"github.com/smart-attendance/shared/middleware"
	"github.com/smart-attendance/shared/response"
)

type AdjustmentHandler struct {
	adjService *service.AttendanceAdjustmentService
}

func NewAdjustmentHandler(adjService *service.AttendanceAdjustmentService) *AdjustmentHandler {
	return &AdjustmentHandler{
		adjService: adjService,
	}
}

// GetMyRequests handles GET /api/adjustments/my
func (h *AdjustmentHandler) GetMyRequests(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	result, err := h.adjService.GetMyRequests(userID, page, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.JSONList(w, result.Records, result.Page, result.Limit, result.Total)
}

// CreateRequest handles POST /api/adjustments/my
func (h *AdjustmentHandler) CreateRequest(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	var req dto.AdjustmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	err := h.adjService.CreateRequest(userID, req.WorkDate, req.CheckIn, req.CheckOut, req.Reason)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, map[string]string{"status": "created"})
}

// GetBranchRequests handles GET /api/adjustments/manage
func (h *AdjustmentHandler) GetBranchRequests(w http.ResponseWriter, r *http.Request) {
	branchID := middleware.GetBranchID(r)
	status := r.URL.Query().Get("status")
	if status == "" {
		status = string(model.AdjustStatusPending)
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	result, err := h.adjService.GetBranchRequests(branchID, status, page, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.JSONList(w, result.Records, result.Page, result.Limit, result.Total)
}

// ReviewRequest handles POST /api/adjustments/manage/:id/review
func (h *AdjustmentHandler) ReviewRequest(w http.ResponseWriter, r *http.Request) {
	requestID := chi.URLParam(r, "id")
	reviewerID := middleware.GetUserID(r)

	var req dto.AdjustmentReview
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	adjStatus := model.AdjustmentStatus(req.Status)
	if adjStatus != model.AdjustStatusApproved && adjStatus != model.AdjustStatusRejected {
		response.Error(w, http.StatusBadRequest, "INVALID_STATUS", "status must be 'approved' or 'rejected'")
		return
	}

	err := h.adjService.ReviewRequest(reviewerID, requestID, adjStatus, req.Note)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "REVIEW_FAILED", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": string(adjStatus)})
}
