package middleware

import (
	"net/http"
	"strings"

	"github.com/smart-attendance/smart-attendance/internal/models"
)

// RequireRoles returns middleware that restricts access to specified roles.
func RequireRoles(roles ...models.Role) func(http.Handler) http.Handler {
	allowed := make(map[models.Role]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := GetUserRole(r)
			if !allowed[role] {
				if strings.HasPrefix(r.URL.Path, "/api/") {
					http.Error(w, `{"success":false,"error":{"code":"FORBIDDEN","message":"insufficient permissions"}}`, http.StatusForbidden)
					return
				}
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Convenience middleware
func AdminOnly(next http.Handler) http.Handler {
	return RequireRoles(models.RoleAdmin)(next)
}

func ManagerOrAdmin(next http.Handler) http.Handler {
	return RequireRoles(models.RoleAdmin, models.RoleManager)(next)
}
