package handler

import (
	"log"
	"net/http"

	"github.com/smart-attendance/organization-service/internal/repository"
	"github.com/smart-attendance/shared/response"
)

type ShiftHandler struct {
	shiftRepo *repository.ShiftRepository
}

func NewShiftHandler(shiftRepo *repository.ShiftRepository) *ShiftHandler {
	return &ShiftHandler{shiftRepo: shiftRepo}
}

// shiftDTO is the API response shape for a work shift.
type shiftDTO struct {
	ID                   string `json:"id"`
	BranchID             string `json:"branch_id"`
	Name                 string `json:"name"`
	Code                 string `json:"code"`
	StartTime            string `json:"start_time"`
	EndTime              string `json:"end_time"`
	GracePeriodMinutes   int    `json:"grace_period_minutes"`
	LateThresholdMinutes int    `json:"late_threshold_minutes"`
	IsOvernight          bool   `json:"is_overnight"`
	BreakDurationMinutes int    `json:"break_duration_minutes"`
	WorkingDays          string `json:"working_days"`
	Color                string `json:"color"`
	IsDefault            bool   `json:"is_default"`
	IsActive             bool   `json:"is_active"`
}

// ListByBranch handles GET /api/shifts?branch_id=X
func (h *ShiftHandler) ListByBranch(w http.ResponseWriter, r *http.Request) {
	branchID := r.URL.Query().Get("branch_id")
	if branchID == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_BRANCH_ID", "branch_id query parameter is required")
		return
	}

	shifts, err := h.shiftRepo.ListByBranch(branchID)
	if err != nil {
		log.Printf("[org][handler][shift] list by branch failed: branch_id=%s, err=%v", branchID, err)
		response.Error(w, http.StatusInternalServerError, "LIST_FAILED", "failed to list shifts")
		return
	}

	dtos := make([]shiftDTO, len(shifts))
	for i, s := range shifts {
		dtos[i] = shiftDTO{
			ID:                   s.ID,
			BranchID:             s.BranchID,
			Name:                 s.Name,
			Code:                 s.Code,
			StartTime:            s.StartTime,
			EndTime:              s.EndTime,
			GracePeriodMinutes:   s.GracePeriodMinutes,
			LateThresholdMinutes: s.LateThresholdMinutes,
			IsOvernight:          s.IsOvernight,
			BreakDurationMinutes: s.BreakDurationMinutes,
			WorkingDays:          s.WorkingDays,
			Color:                s.Color,
			IsDefault:            s.IsDefault,
			IsActive:             s.IsActive,
		}
	}

	response.JSON(w, http.StatusOK, dtos)
}

// FindUserShift handles GET /api/internal/shifts/user?user_id=X&branch_id=X&work_date=X
// Internal endpoint used by Attendance service.
func (h *ShiftHandler) FindUserShift(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	branchID := r.URL.Query().Get("branch_id")
	workDate := r.URL.Query().Get("work_date")

	if userID == "" || branchID == "" || workDate == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_PARAMS", "user_id, branch_id, and work_date are required")
		return
	}

	shift, err := h.shiftRepo.FindUserShift(userID, branchID, workDate)
	if err != nil {
		log.Printf("[org][handler][shift] find user shift failed: user_id=%s, branch_id=%s, work_date=%s, err=%v",
			userID, branchID, workDate, err)
		response.Error(w, http.StatusNotFound, "SHIFT_NOT_FOUND", "no active shift found for user")
		return
	}

	response.JSON(w, http.StatusOK, shiftDTO{
		ID:                   shift.ID,
		BranchID:             shift.BranchID,
		Name:                 shift.Name,
		Code:                 shift.Code,
		StartTime:            shift.StartTime,
		EndTime:              shift.EndTime,
		GracePeriodMinutes:   shift.GracePeriodMinutes,
		LateThresholdMinutes: shift.LateThresholdMinutes,
		IsOvernight:          shift.IsOvernight,
		BreakDurationMinutes: shift.BreakDurationMinutes,
		WorkingDays:          shift.WorkingDays,
		Color:                shift.Color,
		IsDefault:            shift.IsDefault,
		IsActive:             shift.IsActive,
	})
}
