package middleware

import (
	"net/http"
	"strings"
)

// CORS returns middleware that handles Cross-Origin Resource Sharing.
// It allows the specified origin, common HTTP methods, and credential headers.
func CORS(allowOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin matches allowed origins
			if origin != "" && isAllowedOrigin(origin, allowOrigin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else if allowOrigin == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Cookie, X-Requested-With")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.Header().Set("Access-Control-Expose-Headers", "X-Request-Id")

			// Handle preflight OPTIONS request
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isAllowedOrigin checks if the request origin matches the configured allowed origin(s).
// Supports comma-separated list of origins.
func isAllowedOrigin(origin, allowed string) bool {
	if allowed == "*" {
		return true
	}
	for _, o := range strings.Split(allowed, ",") {
		if strings.TrimSpace(o) == origin {
			return true
		}
	}
	return false
}
