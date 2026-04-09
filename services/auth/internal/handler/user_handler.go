package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/smart-attendance/auth-service/internal/repository"
	"github.com/smart-attendance/auth-service/internal/service"
	"github.com/smart-attendance/shared/dto"
	"github.com/smart-attendance/shared/middleware"
	"github.com/smart-attendance/shared/response"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// List handles GET /api/users
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	params := repository.UserListParams{
		Page:     page,
		Limit:    limit,
		BranchID: r.URL.Query().Get("branch_id"),
		Role:     r.URL.Query().Get("role"),
		Search:   r.URL.Query().Get("search"),
	}

	if isActiveStr := r.URL.Query().Get("is_active"); isActiveStr != "" {
		isActive := isActiveStr == "true" || isActiveStr == "1"
		params.IsActive = &isActive
	}

	result, err := h.userService.List(params)
	if err != nil {
		log.Printf("[auth][handler][user] list failed: %v", err)
		response.Error(w, http.StatusInternalServerError, "LIST_FAILED", "failed to list users")
		return
	}

	// Convert to DTOs
	users := make([]dto.User, len(result.Users))
	for i, u := range result.Users {
		users[i] = dto.User{
			ID:        u.ID,
			Email:     u.Email,
			FullName:  u.FullName,
			Phone:     u.Phone,
			Role:      string(u.Role),
			BranchID:  u.BranchID,
			IsActive:  u.IsActive,
			CreatedAt: u.CreatedAt,
		}
	}

	response.JSONList(w, users, page, limit, result.Total)
}

// Create handles POST /api/users
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input service.CreateUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if input.Email == "" || input.Password == "" || input.FullName == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_FIELDS", "email, password, and full_name are required")
		return
	}

	if input.Role == "" {
		input.Role = "employee"
	}

	user, err := h.userService.Create(input)
	if err != nil {
		log.Printf("[auth][handler][user] create failed: email=%s, err=%v", input.Email, err)
		if err.Error() == "email already exists" {
			response.Error(w, http.StatusConflict, "DUPLICATE_EMAIL", err.Error())
			return
		}
		response.Error(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, dto.User{
		ID:        user.ID,
		Email:     user.Email,
		FullName:  user.FullName,
		Phone:     user.Phone,
		Role:      string(user.Role),
		BranchID:  user.BranchID,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
	})
}

// GetByID handles GET /api/users/{id}
func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ID", "user id is required")
		return
	}

	user, err := h.userService.GetByID(id)
	if err != nil {
		log.Printf("[auth][handler][user] get by id failed: id=%s, err=%v", id, err)
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}

	response.JSON(w, http.StatusOK, dto.User{
		ID:        user.ID,
		Email:     user.Email,
		FullName:  user.FullName,
		Phone:     user.Phone,
		Role:      string(user.Role),
		BranchID:  user.BranchID,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
	})
}

// Update handles PUT /api/users/{id}
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ID", "user id is required")
		return
	}

	var input service.UpdateUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	user, err := h.userService.Update(id, input)
	if err != nil {
		log.Printf("[auth][handler][user] update failed: id=%s, err=%v", id, err)
		if err.Error() == "email already exists" {
			response.Error(w, http.StatusConflict, "DUPLICATE_EMAIL", err.Error())
			return
		}
		response.Error(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, dto.User{
		ID:        user.ID,
		Email:     user.Email,
		FullName:  user.FullName,
		Phone:     user.Phone,
		Role:      string(user.Role),
		BranchID:  user.BranchID,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
	})
}

// Delete handles DELETE /api/users/{id}
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ID", "user id is required")
		return
	}

	if err := h.userService.Delete(id); err != nil {
		log.Printf("[auth][handler][user] delete failed: id=%s, err=%v", id, err)
		response.Error(w, http.StatusInternalServerError, "DELETE_FAILED", "failed to delete user")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "user deleted successfully"})
}

// GetInternal handles GET /api/internal/users/{id}
// Internal endpoint used by other services (no RBAC check).
func (h *UserHandler) GetInternal(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ID", "user id is required")
		return
	}

	user, err := h.userService.GetByID(id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}

	response.JSON(w, http.StatusOK, dto.User{
		ID:        user.ID,
		Email:     user.Email,
		FullName:  user.FullName,
		Phone:     user.Phone,
		Role:      string(user.Role),
		BranchID:  user.BranchID,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
	})
}

// Profile handles GET /api/profile
// Returns the current user's profile based on X-User-ID header from the gateway.
func (h *UserHandler) Profile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	if userID == "" {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return
	}

	user, err := h.userService.GetByID(userID)
	if err != nil {
		log.Printf("[auth][handler][user] profile failed: user_id=%s, err=%v", userID, err)
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}

	response.JSON(w, http.StatusOK, dto.User{
		ID:        user.ID,
		Email:     user.Email,
		FullName:  user.FullName,
		Phone:     user.Phone,
		Role:      string(user.Role),
		BranchID:  user.BranchID,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
	})
}
