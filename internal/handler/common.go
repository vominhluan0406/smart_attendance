package handler

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/smart-attendance/smart-attendance/internal/middleware"
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
