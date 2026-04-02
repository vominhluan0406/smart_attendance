package handler

import (
	"log"
	"net/http"
	"strings"

	"github.com/smart-attendance/smart-attendance/internal/middleware"
	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/renderer"
	"github.com/smart-attendance/smart-attendance/internal/service"
)

type HomeHandler struct {
	render        *renderer.Renderer
	branchService *service.BranchService
}

func NewHomeHandler(render *renderer.Renderer, branchService *service.BranchService) *HomeHandler {
	return &HomeHandler{render: render, branchService: branchService}
}

func (h *HomeHandler) Index(w http.ResponseWriter, r *http.Request) {
	data := userContext(r)

	// Check which methods are enabled for user's branch
	branchID := middleware.GetBranchID(r)
	if branchID != "" {
		if branch, err := h.branchService.GetByIDCached(branchID); err == nil {
			data["QREnabled"] = h.branchService.HasMethod(branch, models.MethodQRTOTP)
			data["FaceEnabled"] = h.branchService.HasMethod(branch, models.MethodFace)
			data["PasswordEnabled"] = h.branchService.HasMethod(branch, models.MethodPassword)
		}
	}

	if err := h.render.Render(w, "home.html", data); err != nil {
		log.Printf("[handler][home] render error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *HomeHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

// NotFound renders the 404 error page.
func (h *HomeHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"success":false,"error":{"code":"NOT_FOUND","message":"endpoint not found"}}`))
		return
	}
	w.WriteHeader(http.StatusNotFound)
	h.render.Render(w, "error.html", map[string]interface{}{"Code": 404})
}

// Forbidden renders the 403 error page.
func (h *HomeHandler) Forbidden(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"success":false,"error":{"code":"FORBIDDEN","message":"insufficient permissions"}}`))
		return
	}
	w.WriteHeader(http.StatusForbidden)
	h.render.Render(w, "error.html", map[string]interface{}{"Code": 403})
}

// InternalError renders the 500 error page.
func (h *HomeHandler) InternalError(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"success":false,"error":{"code":"INTERNAL_ERROR","message":"internal server error"}}`))
		return
	}
	w.WriteHeader(http.StatusInternalServerError)
	h.render.Render(w, "error.html", map[string]interface{}{"Code": 500})
}
