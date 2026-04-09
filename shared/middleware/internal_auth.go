package middleware

import "net/http"

// Headers injected by API Gateway after JWT validation.
const (
	HeaderUserID     = "X-User-ID"
	HeaderUserRole   = "X-User-Role"
	HeaderUserEmail  = "X-User-Email"
	HeaderBranchID   = "X-Branch-ID"
	HeaderBranchName = "X-Branch-Name"
)

// GetUserID extracts user ID from internal auth headers.
func GetUserID(r *http.Request) string {
	return r.Header.Get(HeaderUserID)
}

func GetUserRole(r *http.Request) string {
	return r.Header.Get(HeaderUserRole)
}

func GetUserEmail(r *http.Request) string {
	return r.Header.Get(HeaderUserEmail)
}

func GetBranchID(r *http.Request) string {
	return r.Header.Get(HeaderBranchID)
}

func GetBranchName(r *http.Request) string {
	return r.Header.Get(HeaderBranchName)
}

// RequireInternalAuth rejects requests without X-User-ID header (must come from Gateway).
func RequireInternalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetUserID(r) == "" {
			http.Error(w, `{"success":false,"error":{"code":"UNAUTHORIZED","message":"missing internal auth"}}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole returns middleware that checks if X-User-Role matches any allowed role.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !allowed[GetUserRole(r)] {
				http.Error(w, `{"success":false,"error":{"code":"FORBIDDEN","message":"insufficient permissions"}}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
