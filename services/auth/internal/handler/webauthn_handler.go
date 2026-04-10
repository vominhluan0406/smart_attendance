package handler

import (
	"net/http"

	"github.com/smart-attendance/auth-service/internal/service"
	"github.com/smart-attendance/shared/response"
)

type WebAuthnHandler struct {
	webService *service.WebAuthnService
}

func NewWebAuthnHandler(webService *service.WebAuthnService) *WebAuthnHandler {
	return &WebAuthnHandler{webService: webService}
}

// BeginRegistration handles GET /api/webauthn/register/begin
func (h *WebAuthnHandler) BeginRegistration(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "X-User-ID header required")
		return
	}

	options, err := h.webService.BeginRegistration(userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "REGISTRATION_START_FAILED", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, options)
}

// FinishRegistration handles POST /api/webauthn/register/finish
func (h *WebAuthnHandler) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "X-User-ID header required")
		return
	}

	err := h.webService.FinishRegistration(userID, r)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "REGISTRATION_FINISH_FAILED", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "credential registered successfully"})
}
