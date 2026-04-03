package handler

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/smart-attendance/smart-attendance/internal/middleware"
	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/service"
)

// userContext returns common template data from JWT claims.
func userContext(r *http.Request) map[string]interface{} {
	return map[string]interface{}{
		"UserRole":   middleware.GetUserRole(r),
		"UserBranch": middleware.GetBranchID(r),
		"UserName":   middleware.GetFullName(r),
		"BranchName": middleware.GetBranchName(r),
	}
}

// injectBranchFlags adds QREnabled, PasswordEnabled, FaceEnabled etc. to template data.
// Call this after userContext() in any handler that renders a full page with nav.
func injectBranchFlags(data map[string]interface{}, r *http.Request, branchService *service.BranchService) {
	branchID := middleware.GetBranchID(r)
	if branchID == "" {
		return
	}
	branch, err := branchService.GetByIDCached(branchID)
	if err != nil {
		return
	}
	data["QREnabled"] = branchService.HasMethod(branch, models.MethodQRTOTP)
	data["FaceEnabled"] = branchService.HasMethod(branch, models.MethodFace)
	data["PasswordEnabled"] = branchService.HasMethod(branch, models.MethodPassword)
	data["WiFiGPSEnabled"] = branchService.HasMethod(branch, models.MethodWiFiGPS)
	data["BiometricRequired"] = branch.RequireBiometric
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func parseLatLng(latStr, lngStr string) (*float64, *float64) {
	var lat, lng *float64
	if latStr != "" {
		if v, err := strconv.ParseFloat(latStr, 64); err == nil {
			lat = &v
		}
	}
	if lngStr != "" {
		if v, err := strconv.ParseFloat(lngStr, 64); err == nil {
			lng = &v
		}
	}
	return lat, lng
}

func parseOptionalFloat(s string) *float64 {
	if s == "" {
		return nil
	}
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return &v
	}
	return nil
}

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
