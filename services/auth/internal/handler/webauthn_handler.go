package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/smart-attendance/auth-service/internal/repository"
	"github.com/smart-attendance/auth-service/internal/service"
	"github.com/smart-attendance/shared/response"
)

type WebAuthnHandler struct {
	webService *service.WebAuthnService
	credRepo   *repository.CredentialRepository
}

func NewWebAuthnHandler(webService *service.WebAuthnService, credRepo *repository.CredentialRepository) *WebAuthnHandler {
	return &WebAuthnHandler{webService: webService, credRepo: credRepo}
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

// ListCredentials handles GET /api/users/{id}/credentials — admin list user's credentials
func (h *WebAuthnHandler) ListCredentials(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	creds, err := h.credRepo.ListByUserID(userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "FETCH_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, creds)
}

// ApproveCredential handles POST /api/users/{id}/credentials/{credId}/approve
func (h *WebAuthnHandler) ApproveCredential(w http.ResponseWriter, r *http.Request) {
	credID := chi.URLParam(r, "credId")
	cred, err := h.credRepo.FindByID(credID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Không tìm thấy credential")
		return
	}
	cred.IsApproved = true
	if err := h.credRepo.Update(cred); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "Đã phê duyệt"})
}

// RevokeCredential handles POST /api/users/{id}/credentials/{credId}/revoke
func (h *WebAuthnHandler) RevokeCredential(w http.ResponseWriter, r *http.Request) {
	credID := chi.URLParam(r, "credId")
	cred, err := h.credRepo.FindByID(credID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Không tìm thấy credential")
		return
	}
	cred.IsApproved = false
	if err := h.credRepo.Update(cred); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "Đã thu hồi"})
}

// DeleteCredential handles DELETE /api/users/{id}/credentials/{credId}
func (h *WebAuthnHandler) DeleteCredential(w http.ResponseWriter, r *http.Request) {
	credID := chi.URLParam(r, "credId")
	if err := h.credRepo.Delete(credID); err != nil {
		response.Error(w, http.StatusInternalServerError, "DELETE_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "Đã xoá credential"})
}
