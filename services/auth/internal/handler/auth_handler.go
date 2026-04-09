package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/smart-attendance/auth-service/internal/service"
	"github.com/smart-attendance/shared/dto"
	"github.com/smart-attendance/shared/response"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Login handles POST /api/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[auth][handler][auth] login: invalid request body: %v", err)
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_FIELDS", "email and password are required")
		return
	}

	ipAddress := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ipAddress = strings.Split(forwarded, ",")[0]
	}
	userAgent := r.Header.Get("User-Agent")

	result, err := h.authService.Login(req.Email, req.Password, ipAddress, userAgent)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "AUTH_FAILED", err.Error())
		return
	}

	loginResp := dto.LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		User: dto.User{
			ID:       result.User.ID,
			Email:    result.User.Email,
			FullName: result.User.FullName,
			Phone:    result.User.Phone,
			Role:     string(result.User.Role),
			BranchID: result.User.BranchID,
			IsActive: result.User.IsActive,
		},
	}

	response.JSON(w, http.StatusOK, loginResp)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type refreshResponse struct {
	AccessToken string `json:"access_token"`
}

// Refresh handles POST /api/auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_FIELDS", "refresh_token is required")
		return
	}

	accessToken, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "REFRESH_FAILED", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, refreshResponse{AccessToken: accessToken})
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Logout handles POST /api/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_FIELDS", "refresh_token is required")
		return
	}

	if err := h.authService.Logout(req.RefreshToken); err != nil {
		response.Error(w, http.StatusInternalServerError, "LOGOUT_FAILED", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "logged out successfully"})
}

// ValidateToken handles GET /api/internal/validate-token
// Used by the API Gateway to validate JWT tokens.
func (h *AuthHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		response.Error(w, http.StatusUnauthorized, "MISSING_TOKEN", "Authorization header is required")
		return
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		response.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", "Bearer token required")
		return
	}

	claims, err := h.authService.ValidateToken(tokenString)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "INVALID_TOKEN", err.Error())
		return
	}

	validation := dto.TokenValidation{
		UserID:   claims.UserID,
		Email:    claims.Email,
		FullName: claims.FullName,
		Role:     claims.Role,
		BranchID: claims.BranchID,
	}

	response.JSON(w, http.StatusOK, validation)
}
