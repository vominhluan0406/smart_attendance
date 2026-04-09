package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/smart-attendance/attendance-service/internal/repository"
	"github.com/smart-attendance/attendance-service/internal/service"
	"github.com/smart-attendance/shared/dto"
	"github.com/smart-attendance/shared/response"
)

// InternalHandler exposes internal endpoints for other microservices.
type InternalHandler struct {
	attendanceService *service.AttendanceService
}

func NewInternalHandler(attendanceService *service.AttendanceService) *InternalHandler {
	return &InternalHandler{
		attendanceService: attendanceService,
	}
}

// ListAttendance handles GET /api/internal/attendance
// Called by Analytics service to query attendance records.
func (h *InternalHandler) ListAttendance(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	params := repository.AttendanceListParams{
		Page:     page,
		Limit:    limit,
		UserID:   q.Get("user_id"),
		BranchID: q.Get("branch_id"),
		Status:   q.Get("status"),
	}

	if df := q.Get("date_from"); df != "" {
		if t, err := time.Parse("2006-01-02", df); err == nil {
			params.DateFrom = &t
		}
	}
	if dt := q.Get("date_to"); dt != "" {
		if t, err := time.Parse("2006-01-02", dt); err == nil {
			params.DateTo = &t
		}
	}

	result, err := h.attendanceService.List(params)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.JSONList(w, result.Records, result.Page, result.Limit, result.Total)
}

// SyncLeave handles POST /api/internal/attendance/sync-leave
// Called by Leave service after a leave request is approved.
// Creates attendance records with status "leave" for each day in the leave range.
func (h *InternalHandler) SyncLeave(w http.ResponseWriter, r *http.Request) {
	var req dto.SyncLeaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	if req.UserID == "" || req.BranchID == "" || req.StartDate == "" || req.EndDate == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_FIELDS", "user_id, branch_id, start_date, end_date are required")
		return
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_DATE", "invalid start_date format")
		return
	}
	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_DATE", "invalid end_date format")
		return
	}

	created := 0
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		workDate := d.Format("2006-01-02")
		note := req.Note
		if note == "" {
			note = "Leave (synced)"
		}

		if err := h.attendanceService.CreateLeaveAttendance(req.UserID, req.BranchID, workDate, note); err != nil {
			log.Printf("[handler][internal] sync-leave error: user=%s date=%s err=%v", req.UserID, workDate, err)
			continue
		}
		created++
	}

	log.Printf("[handler][internal] sync-leave completed: user=%s dates=%s~%s created=%d",
		req.UserID, req.StartDate, req.EndDate, created)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"created": created,
	})
}
