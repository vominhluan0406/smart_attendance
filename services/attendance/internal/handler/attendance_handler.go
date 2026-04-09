package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/smart-attendance/attendance-service/internal/client"
	"github.com/smart-attendance/attendance-service/internal/repository"
	"github.com/smart-attendance/attendance-service/internal/service"
	"github.com/smart-attendance/shared/middleware"
	"github.com/smart-attendance/shared/response"
)

type AttendanceHandler struct {
	attendanceService *service.AttendanceService
	totpService       *service.TOTPService
	orgClient         *client.OrgClient
}

func NewAttendanceHandler(
	attendanceService *service.AttendanceService,
	totpService *service.TOTPService,
	orgClient *client.OrgClient,
) *AttendanceHandler {
	return &AttendanceHandler{
		attendanceService: attendanceService,
		totpService:       totpService,
		orgClient:         orgClient,
	}
}

// APILogTime handles POST /api/attendance/log
func (h *AttendanceHandler) APILogTime(w http.ResponseWriter, r *http.Request) {
	var input service.LogTimeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}
	input.UserID = middleware.GetUserID(r)
	input.IP = getClientIP(r)
	input.UserAgent = r.UserAgent()

	result, err := h.attendanceService.LogTime(input)
	if err != nil {
		log.Printf("[handler][attendance] log-time failed for user %s: %v", input.UserID, err)
		response.Error(w, http.StatusBadRequest, "LOG_TIME_FAILED", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// APICheckProximity handles POST /api/attendance/check-proximity
func (h *AttendanceHandler) APICheckProximity(w http.ResponseWriter, r *http.Request) {
	var input service.LogTimeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}
	input.UserID = middleware.GetUserID(r)
	input.IP = getClientIP(r)

	result, err := h.attendanceService.CheckProximity(input)
	if err != nil {
		log.Printf("[handler][attendance] check-proximity failed for user %s: %v", input.UserID, err)
		response.Error(w, http.StatusBadRequest, "CHECK_FAILED", err.Error())
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// APIStatus handles GET /api/attendance/status
func (h *AttendanceHandler) APIStatus(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	att, err := h.attendanceService.GetTodayStatus(userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	logs, _ := h.attendanceService.GetTodayLogs(userID)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"attendance": att,
		"logs":       logs,
	})
}

// APIList handles GET /api/attendance with pagination and filters
func (h *AttendanceHandler) APIList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	params := repository.AttendanceListParams{
		Page:     page,
		Limit:    limit,
		UserID:   q.Get("user_id"),
		BranchID: q.Get("branch_id"),
		Status:   q.Get("status"),
	}

	if df := q.Get("date_from"); df != "" {
		if t, err := time.Parse("2006-01-02", df); err == nil {
			params.DateFrom = &t
		}
	}
	if dt := q.Get("date_to"); dt != "" {
		if t, err := time.Parse("2006-01-02", dt); err == nil {
			params.DateTo = &t
		}
	}

	result, err := h.attendanceService.List(params)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	response.JSONList(w, result.Records, result.Page, result.Limit, result.Total)
}

// QRCode handles GET /api/attendance/qr/:branchId/code -- returns JSON with TOTP code
func (h *AttendanceHandler) QRCode(w http.ResponseWriter, r *http.Request) {
	branchID := chi.URLParam(r, "branchId")

	branch, err := h.orgClient.GetBranch(branchID)
	if err != nil {
		log.Printf("[handler][attendance] QR code error: branch %s not found: %v", branchID, err)
		response.Error(w, http.StatusNotFound, "BRANCH_NOT_FOUND", "branch not found")
		return
	}

	code, remaining, err := h.totpService.GenerateCode(branch.TOTPSecret)
	if err != nil {
		log.Printf("[handler][attendance] TOTP generate error for branch %s: %v", branchID, err)
		response.Error(w, http.StatusInternalServerError, "TOTP_ERROR", "failed to generate code")
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"branch_id": branchID,
		"code":      code,
		"remaining": remaining,
	})
}

// QRImage handles GET /api/attendance/qr/:branchId/image -- returns QR code PNG
func (h *AttendanceHandler) QRImage(w http.ResponseWriter, r *http.Request) {
	branchID := chi.URLParam(r, "branchId")

	branch, err := h.orgClient.GetBranch(branchID)
	if err != nil {
		http.Error(w, "Branch not found", http.StatusNotFound)
		return
	}

	code, _, err := h.totpService.GenerateCode(branch.TOTPSecret)
	if err != nil {
		http.Error(w, "Failed to generate code", http.StatusInternalServerError)
		return
	}

	qrContent := branchID + ":" + code
	png, err := qrcode.Encode(qrContent, qrcode.Medium, 256)
	if err != nil {
		log.Printf("[handler][attendance] QR encode error: %v", err)
		http.Error(w, "Failed to generate QR", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write(png)
}

// --- Helpers ---

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For first (from proxy/load balancer)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take first IP in chain
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fallback to RemoteAddr (strip port)
	addr := r.RemoteAddr
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}
