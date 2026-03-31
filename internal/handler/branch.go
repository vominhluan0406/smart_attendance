package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/smart-attendance/smart-attendance/internal/middleware"
	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/renderer"
	"github.com/smart-attendance/smart-attendance/internal/repository"
	"github.com/smart-attendance/smart-attendance/internal/service"
)

type BranchHandler struct {
	branchService *service.BranchService
	render        *renderer.Renderer
}

func NewBranchHandler(branchService *service.BranchService, render *renderer.Renderer) *BranchHandler {
	return &BranchHandler{branchService: branchService, render: render}
}

// --- HTMX Pages ---

func (h *BranchHandler) ListPage(w http.ResponseWriter, r *http.Request) {
	params := h.parseListParams(r)
	result, err := h.branchService.List(params)
	if err != nil {
		log.Printf("[handler][branch] list error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hasNext := int64(result.Page*result.Limit) < result.Total
	data := map[string]interface{}{
		"Branches":    result.Branches,
		"Total":       result.Total,
		"Page":        result.Page,
		"Limit":       result.Limit,
		"Search":      params.Search,
		"HasNextPage": hasNext,
		"HasPrevPage": result.Page > 1,
		"NextPage":    result.Page + 1,
		"PrevPage":    result.Page - 1,
	}

	data["UserRole"] = middleware.GetUserRole(r)
	data["UserBranch"] = middleware.GetBranchID(r)

	if r.Header.Get("HX-Request") == "true" && r.Header.Get("HX-Boosted") != "true" {
		h.render.RenderPartial(w, "branch_list.html", data)
		return
	}
	h.render.Render(w, "branches.html", data)
}

func (h *BranchHandler) CreatePage(w http.ResponseWriter, r *http.Request) {
	h.render.Render(w, "branch_create.html", map[string]interface{}{
		"UserRole":   middleware.GetUserRole(r),
		"UserBranch": middleware.GetBranchID(r),
	})
}

func (h *BranchHandler) EditPage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	branch, err := h.branchService.GetByID(id)
	if err != nil {
		log.Printf("[handler][branch] edit page error: %v", err)
		http.Error(w, "Branch not found", http.StatusNotFound)
		return
	}

	empCount, _ := h.branchService.GetEmployeeCount(id)

	h.render.Render(w, "branch_edit.html", map[string]interface{}{
		"Branch":        branch,
		"EmployeeCount": empCount,
		"UserRole":      middleware.GetUserRole(r),
		"UserBranch":    middleware.GetBranchID(r),
	})
}

// --- HTMX Form Handlers ---

func (h *BranchHandler) CreateForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.render.RenderPartial(w, "auth_error.html", "Invalid form data")
		return
	}

	lat, lng := parseLatLng(r.FormValue("lat"), r.FormValue("lng"))
	radiusM, _ := strconv.Atoi(r.FormValue("radius_m"))

	methods := r.Form["allowed_methods"]

	input := service.CreateBranchInput{
		Name:           r.FormValue("name"),
		Address:        r.FormValue("address"),
		Lat:            lat,
		Lng:            lng,
		RadiusM:        radiusM,
		AllowedMethods: strings.Join(methods, ","),
		WorkStartTime:  r.FormValue("work_start_time"),
		WorkEndTime:    r.FormValue("work_end_time"),
	}

	if _, err := h.branchService.Create(input); err != nil {
		log.Printf("[handler][branch] create error: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		h.render.RenderPartial(w, "auth_error.html", "Failed to create branch")
		return
	}

	w.Header().Set("HX-Redirect", "/branches")
	w.WriteHeader(http.StatusOK)
}

func (h *BranchHandler) UpdateForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.render.RenderPartial(w, "auth_error.html", "Invalid form data")
		return
	}

	lat, lng := parseLatLng(r.FormValue("lat"), r.FormValue("lng"))
	radiusM, _ := strconv.Atoi(r.FormValue("radius_m"))
	isActive := r.FormValue("is_active") == "on"
	methods := r.Form["allowed_methods"]

	input := service.UpdateBranchInput{
		Name:           r.FormValue("name"),
		Address:        r.FormValue("address"),
		Lat:            lat,
		Lng:            lng,
		RadiusM:        radiusM,
		AllowedMethods: strings.Join(methods, ","),
		WorkStartTime:  r.FormValue("work_start_time"),
		WorkEndTime:    r.FormValue("work_end_time"),
		IsActive:       &isActive,
	}

	if _, err := h.branchService.Update(id, input); err != nil {
		log.Printf("[handler][branch] update error: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		h.render.RenderPartial(w, "auth_error.html", "Failed to update branch")
		return
	}

	// Update IP whitelist
	ipEntries := parseIPWhitelistForm(r)
	if err := h.branchService.UpdateIPWhitelist(id, ipEntries); err != nil {
		log.Printf("[handler][branch] update IP whitelist error: %v", err)
	}

	// Update locations
	locEntries := parseLocationsForm(r)
	if err := h.branchService.UpdateLocations(id, locEntries); err != nil {
		log.Printf("[handler][branch] update locations error: %v", err)
	}

	w.Header().Set("HX-Redirect", "/branches")
	w.WriteHeader(http.StatusOK)
}

func (h *BranchHandler) DeleteAction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.branchService.Delete(id); err != nil {
		log.Printf("[handler][branch] delete error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		h.render.RenderPartial(w, "auth_error.html", "Failed to delete branch")
		return
	}

	w.Header().Set("HX-Redirect", "/branches")
	w.WriteHeader(http.StatusOK)
}

// --- API JSON Handlers ---

func (h *BranchHandler) APIList(w http.ResponseWriter, r *http.Request) {
	params := h.parseListParams(r)
	result, err := h.branchService.List(params)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    result.Branches,
		"meta": map[string]interface{}{
			"page":  result.Page,
			"limit": result.Limit,
			"total": result.Total,
		},
	})
}

func (h *BranchHandler) APIGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	branch, err := h.branchService.GetByID(id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrBranchNotFound) {
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
		"data":    branch,
	})
}

func (h *BranchHandler) APICreate(w http.ResponseWriter, r *http.Request) {
	var input service.CreateBranchInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": "INVALID_INPUT", "message": "invalid request body"},
		})
		return
	}

	branch, err := h.branchService.Create(input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": "CREATE_FAILED", "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"data":    branch,
	})
}

func (h *BranchHandler) APIUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input service.UpdateBranchInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": "INVALID_INPUT", "message": "invalid request body"},
		})
		return
	}

	branch, err := h.branchService.Update(id, input)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrBranchNotFound) {
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
		"data":    branch,
	})
}

func (h *BranchHandler) APIDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.branchService.Delete(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": "DELETE_FAILED", "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

// --- Helpers ---

func (h *BranchHandler) parseListParams(r *http.Request) repository.BranchListParams {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	return repository.BranchListParams{
		Page:   page,
		Limit:  limit,
		Search: r.URL.Query().Get("search"),
	}
}

func parseLatLng(latStr, lngStr string) (*float64, *float64) {
	if latStr == "" || lngStr == "" {
		return nil, nil
	}
	lat, err1 := strconv.ParseFloat(latStr, 64)
	lng, err2 := strconv.ParseFloat(lngStr, 64)
	if err1 != nil || err2 != nil {
		return nil, nil
	}
	return &lat, &lng
}

func parseIPWhitelistForm(r *http.Request) []models.BranchIPWhitelist {
	cidrs := r.Form["ip_cidr"]
	labels := r.Form["ip_label"]
	var result []models.BranchIPWhitelist
	for i, cidr := range cidrs {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		label := ""
		if i < len(labels) {
			label = strings.TrimSpace(labels[i])
		}
		result = append(result, models.BranchIPWhitelist{IPCIDR: cidr, Label: label})
	}
	return result
}

func parseLocationsForm(r *http.Request) []models.BranchLocation {
	lats := r.Form["loc_lat"]
	lngs := r.Form["loc_lng"]
	radii := r.Form["loc_radius"]
	locLabels := r.Form["loc_label"]
	var result []models.BranchLocation
	for i := range lats {
		lat, err1 := strconv.ParseFloat(strings.TrimSpace(lats[i]), 64)
		if err1 != nil || i >= len(lngs) {
			continue
		}
		lng, err2 := strconv.ParseFloat(strings.TrimSpace(lngs[i]), 64)
		if err2 != nil {
			continue
		}
		radius := 200
		if i < len(radii) {
			if r, err := strconv.Atoi(strings.TrimSpace(radii[i])); err == nil && r > 0 {
				radius = r
			}
		}
		label := ""
		if i < len(locLabels) {
			label = strings.TrimSpace(locLabels[i])
		}
		result = append(result, models.BranchLocation{Lat: lat, Lng: lng, RadiusM: radius, Label: label})
	}
	return result
}
