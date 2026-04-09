package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/smart-attendance/organization-service/internal/model"
	"github.com/smart-attendance/organization-service/internal/repository"
	"github.com/smart-attendance/organization-service/internal/service"
	"github.com/smart-attendance/shared/dto"
	"github.com/smart-attendance/shared/response"
)

type BranchHandler struct {
	branchService *service.BranchService
}

func NewBranchHandler(branchService *service.BranchService) *BranchHandler {
	return &BranchHandler{branchService: branchService}
}

// List handles GET /api/branches
func (h *BranchHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	params := repository.BranchListParams{
		Page:   page,
		Limit:  limit,
		Search: r.URL.Query().Get("search"),
	}

	if isActiveStr := r.URL.Query().Get("is_active"); isActiveStr != "" {
		isActive := isActiveStr == "true" || isActiveStr == "1"
		params.IsActive = &isActive
	}

	result, err := h.branchService.List(params)
	if err != nil {
		log.Printf("[org][handler][branch] list failed: %v", err)
		response.Error(w, http.StatusInternalServerError, "LIST_FAILED", "failed to list branches")
		return
	}

	// Convert to DTOs
	branches := make([]dto.Branch, len(result.Branches))
	for i, b := range result.Branches {
		branches[i] = toBranchDTO(&b)
	}

	response.JSONList(w, branches, result.Page, result.Limit, result.Total)
}

// Create handles POST /api/branches
func (h *BranchHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input service.CreateBranchInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if input.Name == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_FIELDS", "name is required")
		return
	}

	branch, err := h.branchService.Create(input)
	if err != nil {
		log.Printf("[org][handler][branch] create failed: name=%s, err=%v", input.Name, err)
		response.Error(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, toBranchDTO(branch))
}

// GetByID handles GET /api/branches/{id}
func (h *BranchHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ID", "branch id is required")
		return
	}

	branch, err := h.branchService.GetByID(id)
	if err != nil {
		log.Printf("[org][handler][branch] get by id failed: id=%s, err=%v", id, err)
		if err == service.ErrBranchNotFound {
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "branch not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "GET_FAILED", "failed to get branch")
		return
	}

	response.JSON(w, http.StatusOK, toBranchDTOWithRelations(branch))
}

// Update handles PUT /api/branches/{id}
func (h *BranchHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ID", "branch id is required")
		return
	}

	var input service.UpdateBranchInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	branch, err := h.branchService.Update(id, input)
	if err != nil {
		log.Printf("[org][handler][branch] update failed: id=%s, err=%v", id, err)
		if err == service.ErrBranchNotFound {
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "branch not found")
			return
		}
		response.Error(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, toBranchDTO(branch))
}

// Delete handles DELETE /api/branches/{id}
func (h *BranchHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ID", "branch id is required")
		return
	}

	if err := h.branchService.Delete(id); err != nil {
		log.Printf("[org][handler][branch] delete failed: id=%s, err=%v", id, err)
		response.Error(w, http.StatusInternalServerError, "DELETE_FAILED", "failed to delete branch")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "branch deleted successfully"})
}

// GetInternal handles GET /api/internal/branches/{id}
// Internal endpoint used by other services (e.g. Attendance).
// Returns full branch with IP whitelist + locations, cached.
func (h *BranchHandler) GetInternal(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_ID", "branch id is required")
		return
	}

	branch, err := h.branchService.GetByIDCached(id)
	if err != nil {
		if err == service.ErrBranchNotFound {
			response.Error(w, http.StatusNotFound, "NOT_FOUND", "branch not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "GET_FAILED", "failed to get branch")
		return
	}

	response.JSON(w, http.StatusOK, toBranchDTOWithRelations(branch))
}

// --- DTO conversion helpers ---

func toBranchDTO(b *model.Branch) dto.Branch {
	return dto.Branch{
		ID:             b.ID,
		Name:           b.Name,
		Address:        b.Address,
		Lat:            b.Lat,
		Lng:            b.Lng,
		RadiusM:        b.RadiusM,
		AllowedMethods: b.AllowedMethods,
		WorkStartTime:  b.WorkStartTime,
		WorkEndTime:    b.WorkEndTime,
		IsActive:       b.IsActive,
	}
}

func toBranchDTOWithRelations(b *model.Branch) dto.Branch {
	d := toBranchDTO(b)
	d.TOTPSecret = b.TOTPSecret

	if len(b.IPWhitelist) > 0 {
		d.IPWhitelist = make([]dto.IPWhitelist, len(b.IPWhitelist))
		for i, ip := range b.IPWhitelist {
			d.IPWhitelist[i] = dto.IPWhitelist{
				ID:     ip.ID,
				IPCIDR: ip.IPCIDR,
				Label:  ip.Label,
			}
		}
	}

	if len(b.Locations) > 0 {
		d.Locations = make([]dto.BranchLocation, len(b.Locations))
		for i, loc := range b.Locations {
			d.Locations[i] = dto.BranchLocation{
				ID:      loc.ID,
				Label:   loc.Label,
				Lat:     loc.Lat,
				Lng:     loc.Lng,
				RadiusM: loc.RadiusM,
			}
		}
	}

	return d
}
