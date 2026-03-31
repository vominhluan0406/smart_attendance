package handler

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/smart-attendance/smart-attendance/internal/middleware"
	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/renderer"
	"github.com/smart-attendance/smart-attendance/internal/service"
)

type AttendanceHandler struct {
	attendanceService *service.AttendanceService
	branchService     *service.BranchService
	totpService       *service.TOTPService
	userService       *service.UserService
	render            *renderer.Renderer
}

func NewAttendanceHandler(
	attendanceService *service.AttendanceService,
	branchService *service.BranchService,
	totpService *service.TOTPService,
	userService *service.UserService,
	render *renderer.Renderer,
) *AttendanceHandler {
	return &AttendanceHandler{
		attendanceService: attendanceService,
		branchService:     branchService,
		totpService:       totpService,
		userService:       userService,
		render:            render,
	}
}

// ManagerQRRedirect redirects the manager to their branch's QR code page.
func (h *AttendanceHandler) ManagerQRRedirect(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetUserRole(r)
	if role != "manager" && role != "admin" {
		http.Error(w, "Forbidden: Only Manager/Admin can view this page.", http.StatusForbidden)
		return
	}

	userID := middleware.GetUserID(r)
	user, err := h.userService.GetByID(userID)
	if err != nil || user == nil || user.BranchID == nil {
		http.Error(w, "User or Branch not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, "/attendance/qr/"+*user.BranchID, http.StatusFound)
}

// --- HTMX Pages ---

// AttendancePage shows the scanner (always available) + today's log history.
func (h *AttendanceHandler) AttendancePage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	today, _ := h.attendanceService.GetTodayStatus(userID)
	logs, _ := h.attendanceService.GetTodayLogs(userID)

	h.render.Render(w, "attendance.html", map[string]interface{}{
		"Today":      today,
		"Logs":       logs,
		"UserRole":   middleware.GetUserRole(r),
		"UserBranch": middleware.GetBranchID(r),
	})
}

// QRDisplayPage shows the live QR code for a specific branch (Manager/Admin only).
func (h *AttendanceHandler) QRDisplayPage(w http.ResponseWriter, r *http.Request) {
	branchID := chi.URLParam(r, "branchID")

	if middleware.GetUserRole(r) == models.RoleManager {
		if middleware.GetBranchID(r) != branchID {
			http.Error(w, "Forbidden: You can only view QR for your own branch.", http.StatusForbidden)
			return
		}
	}

	branch, err := h.branchService.GetByID(branchID)
	if err != nil {
		log.Printf("[handler][attendance] QR display error: branch %s not found: %v", branchID, err)
		http.Error(w, "Branch not found", http.StatusNotFound)
		return
	}

	code, remaining, err := h.totpService.GenerateCode(branch.TOTPSecret)
	if err != nil {
		log.Printf("[handler][attendance] TOTP generate error for branch %s: %v", branchID, err)
		http.Error(w, "Failed to generate code", http.StatusInternalServerError)
		return
	}

	h.render.Render(w, "qr_display.html", map[string]interface{}{
		"Branch":     branch,
		"BranchID":   branch.ID,
		"TOTPCode":   code,
		"Remaining":  remaining,
		"UserRole":   middleware.GetUserRole(r),
		"UserBranch": middleware.GetBranchID(r),
	})
}

// QRCodePartial returns the QR code partial (HTMX auto-refresh target).
func (h *AttendanceHandler) QRCodePartial(w http.ResponseWriter, r *http.Request) {
	branchID := chi.URLParam(r, "branchID")

	if middleware.GetUserRole(r) == models.RoleManager {
		if middleware.GetBranchID(r) != branchID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	branch, err := h.branchService.GetByIDCached(branchID)
	if err != nil {
		http.Error(w, "Branch not found", http.StatusNotFound)
		return
	}

	code, remaining, err := h.totpService.GenerateCode(branch.TOTPSecret)
	if err != nil {
		http.Error(w, "Failed to generate code", http.StatusInternalServerError)
		return
	}

	h.render.RenderPartial(w, "qr_code.html", map[string]interface{}{
		"BranchID":  branchID,
		"TOTPCode":  code,
		"Remaining": remaining,
	})
}

// QRImage generates a QR code PNG containing the current TOTP code for a branch.
func (h *AttendanceHandler) QRImage(w http.ResponseWriter, r *http.Request) {
	branchID := chi.URLParam(r, "branchID")

	if middleware.GetUserRole(r) == models.RoleManager {
		if middleware.GetBranchID(r) != branchID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	branch, err := h.branchService.GetByIDCached(branchID)
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

// --- HTMX Form Handler ---

// LogTimeForm handles QR scan → log time (replaces separate check-in/check-out).
func (h *AttendanceHandler) LogTimeForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.render.RenderPartial(w, "auth_error.html", "Invalid form data")
		return
	}

	userID := middleware.GetUserID(r)
	lat, lng := parseLatLng(r.FormValue("lat"), r.FormValue("lng"))

	input := service.LogTimeInput{
		UserID:   userID,
		TOTPCode: r.FormValue("totp_code"),
		IP:       getClientIP(r),
		Lat:      lat,
		Lng:      lng,
	}

	result, err := h.attendanceService.LogTime(input)
	if err != nil {
		log.Printf("[handler][attendance] log-time failed for user %s: %v", userID, err)
		w.WriteHeader(http.StatusBadRequest)
		h.render.RenderPartial(w, "checkin_result.html", map[string]interface{}{
			"Success": false,
			"Error":   err.Error(),
		})
		return
	}

	h.render.RenderPartial(w, "checkin_result.html", map[string]interface{}{
		"Success":    true,
		"Attendance": result.Attendance,
		"LogCount":   result.LogCount,
		"TOTP":       result.TOTPVerified,
		"IP":         result.IPVerified,
		"Location":   result.LocVerified,
	})
}

// --- API JSON Handlers ---

// APILogTime handles POST /api/v1/attendance/log
func (h *AttendanceHandler) APILogTime(w http.ResponseWriter, r *http.Request) {
	var input service.LogTimeInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": "INVALID_INPUT", "message": "invalid request body"},
		})
		return
	}
	input.UserID = middleware.GetUserID(r)
	input.IP = getClientIP(r)

	result, err := h.attendanceService.LogTime(input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": "LOG_TIME_FAILED", "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    result,
	})
}

func (h *AttendanceHandler) APIStatus(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	att, err := h.attendanceService.GetTodayStatus(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
		return
	}

	logs, _ := h.attendanceService.GetTodayLogs(userID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    att,
		"logs":    logs,
	})
}

// --- Helpers ---

func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-Ip"); ip != "" {
		return ip
	}
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.Split(forwarded, ",")[0]
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
