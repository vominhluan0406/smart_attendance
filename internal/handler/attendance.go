package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/smart-attendance/smart-attendance/internal/middleware"
	"github.com/smart-attendance/smart-attendance/internal/renderer"
	"github.com/smart-attendance/smart-attendance/internal/service"
)

type AttendanceHandler struct {
	attendanceService *service.AttendanceService
	branchService     *service.BranchService
	totpService       *service.TOTPService
	render            *renderer.Renderer
}

func NewAttendanceHandler(
	attendanceService *service.AttendanceService,
	branchService *service.BranchService,
	totpService *service.TOTPService,
	render *renderer.Renderer,
) *AttendanceHandler {
	return &AttendanceHandler{
		attendanceService: attendanceService,
		branchService:     branchService,
		totpService:       totpService,
		render:            render,
	}
}

// --- HTMX Pages ---

func (h *AttendanceHandler) CheckInPage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	today, _ := h.attendanceService.GetTodayStatus(userID)

	h.render.Render(w, "attendance.html", map[string]interface{}{
		"Today": today,
	})
}

// QRDisplayPage shows the live QR code for a specific branch (Manager/Admin only).
func (h *AttendanceHandler) QRDisplayPage(w http.ResponseWriter, r *http.Request) {
	branchID := chi.URLParam(r, "branchID")

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
		"Branch":    branch,
		"TOTPCode":  code,
		"Remaining": remaining,
	})
}

// QRCodePartial returns the QR code partial (HTMX auto-refresh target).
func (h *AttendanceHandler) QRCodePartial(w http.ResponseWriter, r *http.Request) {
	branchID := chi.URLParam(r, "branchID")

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

	// QR content: branch_id:totp_code
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

// --- HTMX Form Handlers ---

func (h *AttendanceHandler) CheckInForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.render.RenderPartial(w, "auth_error.html", "Invalid form data")
		return
	}

	userID := middleware.GetUserID(r)
	lat, lng := parseLatLng(r.FormValue("lat"), r.FormValue("lng"))

	input := service.CheckInInput{
		UserID:   userID,
		TOTPCode: r.FormValue("totp_code"),
		IP:       getClientIP(r),
		Lat:      lat,
		Lng:      lng,
	}

	result, err := h.attendanceService.CheckIn(input)
	if err != nil {
		log.Printf("[handler][attendance] check-in failed for user %s: %v", userID, err)
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
		"TOTP":       result.TOTPVerified,
		"IP":         result.IPVerified,
		"Location":   result.LocVerified,
	})
}

func (h *AttendanceHandler) CheckOutForm(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	att, err := h.attendanceService.CheckOut(userID)
	if err != nil {
		log.Printf("[handler][attendance] check-out failed for user %s: %v", userID, err)
		w.WriteHeader(http.StatusBadRequest)
		h.render.RenderPartial(w, "checkin_result.html", map[string]interface{}{
			"Success": false,
			"Error":   err.Error(),
		})
		return
	}

	h.render.RenderPartial(w, "checkin_result.html", map[string]interface{}{
		"Success":    true,
		"Attendance": att,
		"CheckedOut": true,
	})
}

// --- API JSON Handlers ---

func (h *AttendanceHandler) APICheckIn(w http.ResponseWriter, r *http.Request) {
	var input service.CheckInInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": "INVALID_INPUT", "message": "invalid request body"},
		})
		return
	}
	input.UserID = middleware.GetUserID(r)
	input.IP = getClientIP(r)

	result, err := h.attendanceService.CheckIn(input)
	if err != nil {
		code := http.StatusBadRequest
		errCode := "CHECK_IN_FAILED"
		if errors.Is(err, service.ErrAlreadyCheckedIn) {
			errCode = "ALREADY_CHECKED_IN"
		}
		writeJSON(w, code, map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": errCode, "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    result,
	})
}

func (h *AttendanceHandler) APICheckOut(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	att, err := h.attendanceService.CheckOut(userID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"error":   map[string]string{"code": "CHECK_OUT_FAILED", "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    att,
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

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"data":       att,
		"checked_in": att != nil,
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
