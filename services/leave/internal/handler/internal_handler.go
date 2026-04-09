package handler

import (
	"log"
	"net/http"

	"github.com/smart-attendance/leave-service/internal/service"
	"github.com/smart-attendance/shared/response"
)

// InternalHandler exposes internal endpoints for other microservices (e.g., Analytics).
type InternalHandler struct {
	leaveService *service.LeaveService
}

func NewInternalHandler(leaveService *service.LeaveService) *InternalHandler {
	return &InternalHandler{
		leaveService: leaveService,
	}
}

// PendingCount handles GET /api/internal/leave/pending-count?branch_id=X
// Returns the count of pending leave requests for a given branch.
// Used by Analytics service.
func (h *InternalHandler) PendingCount(w http.ResponseWriter, r *http.Request) {
	branchID := r.URL.Query().Get("branch_id")
	if branchID == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_BRANCH", "branch_id query parameter is required")
		return
	}

	count, err := h.leaveService.CountPendingByBranch(branchID)
	if err != nil {
		log.Printf("[handler][internal] ERROR PendingCount branch=%s: %v", branchID, err)
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]int64{"count": count})
}
