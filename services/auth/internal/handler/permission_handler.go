package handler

import (
	"log"
	"net/http"

	"github.com/smart-attendance/auth-service/internal/service"
	"github.com/smart-attendance/shared/response"
)

type PermissionHandler struct {
	permService *service.PermissionService
}

func NewPermissionHandler(permService *service.PermissionService) *PermissionHandler {
	return &PermissionHandler{permService: permService}
}

type permissionCheckResponse struct {
	Allowed bool `json:"allowed"`
}

// Check handles GET /api/internal/permissions/check?role=X&code=Y
// Used by the API Gateway and other services to check RBAC.
func (h *PermissionHandler) Check(w http.ResponseWriter, r *http.Request) {
	role := r.URL.Query().Get("role")
	code := r.URL.Query().Get("code")

	if role == "" || code == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_PARAMS", "role and code query parameters are required")
		return
	}

	allowed, err := h.permService.HasPermission(role, code)
	if err != nil {
		log.Printf("[auth][handler][permission] check failed: role=%s, code=%s, err=%v", role, code, err)
		response.Error(w, http.StatusInternalServerError, "CHECK_FAILED", "failed to check permission")
		return
	}

	response.JSON(w, http.StatusOK, permissionCheckResponse{Allowed: allowed})
}
