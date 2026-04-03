package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/smart-attendance/smart-attendance/internal/middleware"
	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/renderer"
	"github.com/smart-attendance/smart-attendance/internal/repository"
	"github.com/smart-attendance/smart-attendance/internal/service"
)

type UserHandler struct {
	userService   *service.UserService
	authService   *service.AuthService
	branchService *service.BranchService
	webauthnService *service.WebAuthnService
	render        *renderer.Renderer
}

func NewUserHandler(userService *service.UserService, authService *service.AuthService, branchService *service.BranchService, webauthnService *service.WebAuthnService, render *renderer.Renderer) *UserHandler {
	return &UserHandler{userService: userService, authService: authService, branchService: branchService, webauthnService: webauthnService, render: render}
}

// --- HTMX Pages ---

func (h *UserHandler) ListPage(w http.ResponseWriter, r *http.Request) {
	params := h.parseListParams(r)
	result, err := h.userService.List(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hasNext := int64(result.Page*result.Limit) < result.Total
	hasPrev := result.Page > 1

	data := map[string]interface{}{
		"Users":       result.Users,
		"Total":       result.Total,
		"Page":        result.Page,
		"Limit":       result.Limit,
		"Search":      params.Search,
		"Role":        params.Role,
		"HasNextPage": hasNext,
		"HasPrevPage": hasPrev,
		"NextPage":    result.Page + 1,
		"PrevPage":    result.Page - 1,
	}

	for k, v := range userContext(r) {
		data[k] = v
	}

	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Boosted") != "true" {
		h.render.RenderPartial(w, "user_list.html", data)
		return
	}
	h.render.Render(w, "users.html", data)
}

func (h *UserHandler) CreatePage(w http.ResponseWriter, r *http.Request) {
	branches, _ := h.branchService.ListAll()
	data := userContext(r)
	data["Branches"] = branches
	h.render.Render(w, "user_create.html", data)
}

func (h *UserHandler) EditPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user, err := h.userService.GetByID(id)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	branches, _ := h.branchService.ListAll()
	currentBranch := ""
	if user.BranchID != nil {
		currentBranch = *user.BranchID
	}
	data := userContext(r)
	data["User"] = user
	data["Branches"] = branches
	data["CurrentBranch"] = currentBranch
	h.render.Render(w, "user_edit.html", data)
}

func (h *UserHandler) ProfilePage(w http.ResponseWriter, r *http.Request) {
	// RBAC: Admin, Manager, Manager Device don't have personal profiles
	role := middleware.GetUserRole(r)
	if role == models.RoleAdmin || role == models.RoleManager || role == models.RoleManagerDevice {
		http.Error(w, "Chức năng này chỉ dành cho nhân viên.", http.StatusForbidden)
		return
	}

	userID := middleware.GetUserID(r)
	user, err := h.userService.GetByID(userID)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := userContext(r)
	data["User"] = user
	h.render.Render(w, "profile.html", data)
}

func (h *UserHandler) RegisterBiometricBegin(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	user, err := h.userService.GetByID(userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "User not found"})
		return
	}

	options, err := h.webauthnService.BeginRegistration(user)
	if err != nil {
		log.Printf("[handler][user] webauthn begin failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, options)
}

func (h *UserHandler) RegisterBiometricFinish(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	user, err := h.userService.GetByID(userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "User not found"})
		return
	}

	if err := h.webauthnService.FinishRegistration(user, r); err != nil {
		log.Printf("[handler][user] webauthn finish failed: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *UserHandler) ApproveCredential(w http.ResponseWriter, r *http.Request) {
	credID := chi.URLParam(r, "credID")
	
	// We use the service to approve (or just repo for simplicity if no complex logic)
	// Let's use the repo directly if we have access to it, 
	// but UserHandler doesn't have credRepo. 
	// Wait, UserHandler has webauthnService which has credRepo?
	// No, webauthnService fields are private.
	// I should probably add Approve/Delete to WebAuthnService or use UserService.
	
	// Better: Add these to UserService or WebAuthnService.
	if err := h.webauthnService.ApproveCredential(credID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

func (h *UserHandler) DeleteCredential(w http.ResponseWriter, r *http.Request) {
	credID := chi.URLParam(r, "credID")
	
	if err := h.webauthnService.DeleteCredential(credID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

// --- HTMX Form Handlers ---

func (h *UserHandler) CreateForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.render.RenderPartial(w, "auth_error.html", "Invalid form data")
		return
	}

	input := service.RegisterInput{
		Email:    r.FormValue("email"),
		Password: r.FormValue("password"),
		FullName: r.FormValue("full_name"),
		Role:     models.Role(r.FormValue("role")),
	}
	if branchID := r.FormValue("branch_id"); branchID != "" {
		input.BranchID = &branchID
	}

	if _, err := h.authService.Register(input); err != nil {
		msg := "Failed to create user"
		if errors.Is(err, service.ErrEmailExists) {
			msg = "Email already registered"
		}
		w.WriteHeader(http.StatusBadRequest)
		h.render.RenderPartial(w, "auth_error.html", msg)
		return
	}

	w.Header().Set("HX-Redirect", "/users")
	w.WriteHeader(http.StatusOK)
}

func (h *UserHandler) UpdateForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.render.RenderPartial(w, "auth_error.html", "Invalid form data")
		return
	}

	isActive := r.FormValue("is_active") == "on"
	branchID := r.FormValue("branch_id")
	var branchPtr *string
	if branchID != "" {
		branchPtr = &branchID
	}
	input := service.UpdateUserInput{
		FullName: r.FormValue("full_name"),
		Email:    r.FormValue("email"),
		Role:     models.Role(r.FormValue("role")),
		IsActive: &isActive,
		Password: r.FormValue("password"),
		BranchID: branchPtr,
	}

	if _, err := h.userService.Update(id, input); err != nil {
		msg := "Failed to update user"
		if errors.Is(err, service.ErrEmailExists) {
			msg = "Email already in use"
		}
		w.WriteHeader(http.StatusBadRequest)
		h.render.RenderPartial(w, "auth_error.html", msg)
		return
	}

	w.Header().Set("HX-Redirect", "/users")
	w.WriteHeader(http.StatusOK)
}

func (h *UserHandler) DeleteAction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// Prevent self-delete
	if id == middleware.GetUserID(r) {
		w.WriteHeader(http.StatusBadRequest)
		h.render.RenderPartial(w, "auth_error.html", "Cannot delete your own account")
		return
	}

	if err := h.userService.Delete(id); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.render.RenderPartial(w, "auth_error.html", "Failed to delete user")
		return
	}

	w.Header().Set("HX-Redirect", "/users")
	w.WriteHeader(http.StatusOK)
}

// --- API JSON Handlers ---

func (h *UserHandler) APIList(w http.ResponseWriter, r *http.Request) {
	params := h.parseListParams(r)
	result, err := h.userService.List(params)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    result.Users,
		"meta": map[string]interface{}{
			"page":  result.Page,
			"limit": result.Limit,
			"total": result.Total,
		},
	})
}

func (h *UserHandler) APIGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	user, err := h.userService.GetByID(id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrUserNotFound) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": "NOT_FOUND", "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    user,
	})
}

func (h *UserHandler) APIUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input service.UpdateUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": "INVALID_INPUT", "message": "invalid request body"},
		})
		return
	}

	user, err := h.userService.Update(id, input)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrUserNotFound) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": "UPDATE_FAILED", "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    user,
	})
}

func (h *UserHandler) APIDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.userService.Delete(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": "DELETE_FAILED", "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    nil,
	})
}

// --- Helpers ---

func (h *UserHandler) parseListParams(r *http.Request) repository.UserListParams {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	return repository.UserListParams{
		Page:     page,
		Limit:    limit,
		Search:   r.URL.Query().Get("search"),
		Role:     r.URL.Query().Get("role"),
		BranchID: r.URL.Query().Get("branch_id"),
	}
}
