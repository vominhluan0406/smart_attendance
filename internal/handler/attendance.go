package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/smart-attendance/smart-attendance/internal/middleware"
	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/renderer"
	"github.com/smart-attendance/smart-attendance/internal/repository"
	"github.com/smart-attendance/smart-attendance/internal/service"
)

type AttendanceHandler struct {
	attendanceService *service.AttendanceService
	branchService     *service.BranchService
	totpService       *service.TOTPService
	userService       *service.UserService
	authService       *service.AuthService
	webauthnService   *service.WebAuthnService
	render            *renderer.Renderer
}

func NewAttendanceHandler(
	attendanceService *service.AttendanceService,
	branchService *service.BranchService,
	totpService *service.TOTPService,
	userService *service.UserService,
	authService *service.AuthService,
	webauthnService *service.WebAuthnService,
	render *renderer.Renderer,
) *AttendanceHandler {
	return &AttendanceHandler{
		attendanceService: attendanceService,
		branchService:     branchService,
		totpService:       totpService,
		userService:       userService,
		authService:       authService,
		webauthnService:   webauthnService,
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

	data := userContext(r)
	data["Today"] = today
	data["Logs"] = logs

	// Check which methods are enabled for user's branch
	branchID := middleware.GetBranchID(r)
	if branchID != "" {
		if branch, err := h.branchService.GetByIDCached(branchID); err == nil {
			data["QREnabled"] = h.branchService.HasMethod(branch, models.MethodQRTOTP)
			data["IPEnabled"] = h.branchService.HasMethod(branch, models.MethodIP)
			data["LocationEnabled"] = h.branchService.HasMethod(branch, models.MethodLocation)
			data["FaceEnabled"] = h.branchService.HasMethod(branch, models.MethodFace)
			data["PasswordEnabled"] = h.branchService.HasMethod(branch, models.MethodPassword)
			data["WiFiGPSEnabled"] = h.branchService.HasMethod(branch, models.MethodWiFiGPS)
			data["BiometricRequired"] = branch.RequireBiometric
		}
	}

	h.render.Render(w, "attendance.html", data)
}

// PasswordCheckinPage shows a login form specifically for attendance check-in (shared device).
func (h *AttendanceHandler) PasswordCheckinPage(w http.ResponseWriter, r *http.Request) {
	data := userContext(r)
	role, _ := data["UserRole"].(models.Role)
	
	// Strictly only allow non-employees to see the check-in page
	if role == models.RoleEmployee {
		log.Printf("[handler][attendance] blocking employee from password checkin page")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Fetch employees of this branch for the dropdown
	branchID := middleware.GetBranchID(r)
	if branchID != "" {
		res, err := h.userService.List(repository.UserListParams{
			BranchID: branchID,
			Role:     string(models.RoleEmployee),
			IsActive: func() *bool { b := true; return &b }(),
			Limit:    100,
		})
		if err == nil {
			data["Employees"] = res.Users
		}
	}

	h.render.Render(w, "password_checkin.html", data)
}

func (h *AttendanceHandler) WiFiGPSCheckinPage(w http.ResponseWriter, r *http.Request) {
	data := userContext(r)
	branchID := middleware.GetBranchID(r)
	if branchID != "" {
		if branch, err := h.branchService.GetByIDCached(branchID); err == nil {
			data["BiometricRequired"] = branch.RequireBiometric
		}
	}

	h.render.Render(w, "wifi_gps_checkin.html", data)
}

func (h *AttendanceHandler) BiometricLoginBegin(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	user, err := h.userService.GetByID(userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "User not found"})
		return
	}

	options, err := h.webauthnService.BeginLogin(user)
	if err != nil {
		log.Printf("[handler][attendance] webauthn login begin failed: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, options)
}

func (h *AttendanceHandler) BiometricLoginFinish(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	user, err := h.userService.GetByID(userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "User not found"})
		return
	}

	if err := h.webauthnService.FinishLogin(user, r); err != nil {
		log.Printf("[handler][attendance] webauthn login finish failed: %v", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "biometric_verified": "1"})
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

	recent, _ := h.attendanceService.GetRecentByBranch(branchID, 5)

	data := userContext(r)
	data["Branch"] = branch
	data["BranchID"] = branch.ID
	data["TOTPCode"] = code
	data["Remaining"] = remaining
	data["RecentLogs"] = recent
	h.render.Render(w, "qr_display.html", data)
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
		h.render.RenderPartial(w, "auth_error.html", "Dữ liệu không hợp lệ")
		return
	}

	userID := middleware.GetUserID(r)
	lat, lng := parseLatLng(r.FormValue("lat"), r.FormValue("lng"))

	input := service.LogTimeInput{
		UserID:          middleware.GetUserID(r),
		TOTPCode:        r.FormValue("totp_code"),
		ScannedBranchID: r.FormValue("scanned_branch_id"),
		Lat:             lat,
		Lng:             lng,
		IP:              getClientIP(r),
		FaceVerified:    r.FormValue("face_verified") == "1",
		BiometricVerified: r.FormValue("biometric_verified") == "1",
	}

	log.Printf("[handler][attendance] LogTimeForm input: User=%s, IP=%s, Lat=%v, Lng=%v, BranchID=%s, TOTP=%s",
		input.UserID, input.IP, input.Lat, input.Lng, input.ScannedBranchID, input.TOTPCode)

	result, err := h.attendanceService.LogTime(input)
	if err != nil {
		log.Printf("[handler][attendance] log-time failed for user %s: %v", userID, err)
		h.render.RenderPartial(w, "checkin_result.html", map[string]interface{}{
			"Success": false,
			"Error":   translateAttendanceError(err),
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
		"Face":       result.FaceVerified,
		"NFC":        result.NFCVerified,
		"Password":   result.PasswordVerified,
	})
}

// PasswordLogForm handles check-in via username/password (Fallback method).
func (h *AttendanceHandler) PasswordLogForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.render.RenderPartial(w, "auth_error.html", "Dữ liệu không hợp lệ")
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	// 1. Verify credentials via AuthService
	user, err := h.authService.VerifyPassword(email, password)
	if err != nil {
		h.render.RenderPartial(w, "auth_error.html", "Email hoặc mật khẩu không chính xác")
		return
	}

	input := service.LogTimeInput{
		UserID:           user.ID,
		IP:               getClientIP(r),
		PasswordVerified: true,
	}

	result, err := h.attendanceService.LogTime(input)
	if err != nil {
		h.render.RenderPartial(w, "checkin_result.html", map[string]interface{}{
			"Success": false,
			"Error":   translateAttendanceError(err),
		})
		return
	}

	h.render.RenderPartial(w, "checkin_result.html", map[string]interface{}{
		"Success":    true,
		"Attendance": result.Attendance,
		"LogCount":   result.LogCount,
		"Password":   true,
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

func translateAttendanceError(err error) string {
	switch {
	case errors.Is(err, service.ErrNoBranch):
		return "Bạn chưa được gán chi nhánh. Vui lòng liên hệ quản lý."
	case errors.Is(err, service.ErrTOTPInvalid):
		return "Mã QR không hợp lệ hoặc đã hết hạn. Vui lòng quét lại."
	case errors.Is(err, service.ErrIPNotAllowed):
		return "Bạn không thể chấm công từ mạng này. Vui lòng kết nối WiFi chi nhánh."
	case errors.Is(err, service.ErrLocationOutside):
		return "Bạn đang ngoài khu vực chi nhánh. Vui lòng di chuyển đến gần hơn."
	case errors.Is(err, service.ErrMethodRequired):
		return "Không xác minh được. Vui lòng quét mã QR tại chi nhánh."
	case errors.Is(err, service.ErrUserNotFound):
		return "Phiên đăng nhập hết hạn. Vui lòng đăng nhập lại."
	case errors.Is(err, service.ErrBranchNotFound):
		return "Chi nhánh không tồn tại. Vui lòng liên hệ quản lý."
	default:
		msg := err.Error()
		if strings.Contains(msg, "invalid or expired QR") {
			return "Mã QR đã hết hạn. Vui lòng quét mã mới."
		}
		if strings.Contains(msg, "QR/TOTP code required") {
			return "Vui lòng quét mã QR tại chi nhánh để chấm công."
		}
		if strings.Contains(msg, "GPS location required") {
			return "Không lấy được vị trí GPS. Vui lòng bật định vị và thử lại."
		}
		if strings.Contains(msg, "location outside") {
			return "Bạn đang ngoài khu vực chi nhánh."
		}
		return "Có lỗi xảy ra. Vui lòng thử lại."
	}
}
