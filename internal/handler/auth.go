package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/renderer"
	"github.com/smart-attendance/smart-attendance/internal/service"
)

type AuthHandler struct {
	authService    *service.AuthService
	render         *renderer.Renderer
	microsoftOAuth bool
}

func NewAuthHandler(authService *service.AuthService, render *renderer.Renderer, microsoftOAuth bool) *AuthHandler {
	return &AuthHandler{authService: authService, render: render, microsoftOAuth: microsoftOAuth}
}

// --- Pages ---

func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	h.render.Render(w, "login.html", map[string]interface{}{
		"Error":          r.URL.Query().Get("error"),
		"MicrosoftOAuth": h.microsoftOAuth,
	})
}

func (h *AuthHandler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	h.render.Render(w, "register.html", nil)
}

// --- HTMX Form Handlers ---

func (h *AuthHandler) LoginForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render.RenderPartial(w, "auth_error.html", "Invalid form data")
		return
	}

	input := service.LoginInput{
		Email:    r.FormValue("email"),
		Password: r.FormValue("password"),
	}

	tokens, _, err := h.authService.Login(input)
	if err != nil {
		msg := "Invalid email or password"
		if errors.Is(err, service.ErrAccountDisabled) {
			msg = "Account is disabled"
		}
		w.WriteHeader(http.StatusUnauthorized)
		h.render.RenderPartial(w, "auth_error.html", msg)
		return
	}

	setTokenCookies(w, tokens)
	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) RegisterForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.render.RenderPartial(w, "auth_error.html", "Invalid form data")
		return
	}

	password := r.FormValue("password")
	confirm := r.FormValue("password_confirm")
	if password != confirm {
		w.WriteHeader(http.StatusBadRequest)
		h.render.RenderPartial(w, "auth_error.html", "Passwords do not match")
		return
	}

	if len(password) < 6 {
		w.WriteHeader(http.StatusBadRequest)
		h.render.RenderPartial(w, "auth_error.html", "Password must be at least 6 characters")
		return
	}

	input := service.RegisterInput{
		Email:    r.FormValue("email"),
		Password: password,
		FullName: r.FormValue("full_name"),
		Role:     models.RoleEmployee,
	}

	_, err := h.authService.Register(input)
	if err != nil {
		msg := "Registration failed"
		if errors.Is(err, service.ErrEmailExists) {
			msg = "Email already registered"
		}
		w.WriteHeader(http.StatusBadRequest)
		h.render.RenderPartial(w, "auth_error.html", msg)
		return
	}

	w.Header().Set("HX-Redirect", "/auth/login")
	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/auth/login", http.StatusFound)
}

// --- API JSON Handlers ---

func (h *AuthHandler) APILogin(w http.ResponseWriter, r *http.Request) {
	var input service.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": "INVALID_INPUT", "message": "invalid request body"},
		})
		return
	}

	tokens, user, err := h.authService.Login(input)
	if err != nil {
		code := http.StatusUnauthorized
		errCode := "INVALID_CREDENTIALS"
		if errors.Is(err, service.ErrAccountDisabled) {
			errCode = "ACCOUNT_DISABLED"
			code = http.StatusForbidden
		}
		writeJSON(w, code, map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": errCode, "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"user":   user,
			"tokens": tokens,
		},
	})
}

func (h *AuthHandler) APIRegister(w http.ResponseWriter, r *http.Request) {
	var input service.RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": "INVALID_INPUT", "message": "invalid request body"},
		})
		return
	}

	input.Role = models.RoleEmployee // Public registration is always employee

	user, err := h.authService.Register(input)
	if err != nil {
		code := http.StatusBadRequest
		errCode := "REGISTRATION_FAILED"
		if errors.Is(err, service.ErrEmailExists) {
			errCode = "EMAIL_EXISTS"
			code = http.StatusConflict
		}
		writeJSON(w, code, map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": errCode, "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    user,
	})
}

func (h *AuthHandler) APIRefresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": "INVALID_INPUT", "message": "invalid request body"},
		})
		return
	}

	tokens, err := h.authService.RefreshToken(body.RefreshToken)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": "INVALID_TOKEN", "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    tokens,
	})
}

// --- Helpers ---

func setTokenCookies(w http.ResponseWriter, tokens *service.TokenPair) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    tokens.AccessToken,
		Path:     "/",
		MaxAge:   tokens.ExpiresIn,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokens.RefreshToken,
		Path:     "/",
		MaxAge:   int((168 * time.Hour).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
