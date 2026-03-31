package middleware

import (
	"net/http"
	"strings"

	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/service"
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
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte(`{"success":false,"error":{"code":"FORBIDDEN","message":"insufficient permissions"}}`))
					return
				}
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(errorPage403))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

const errorPage403 = `<!DOCTYPE html>
<html lang="vi"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>403 — Smart Attendance</title>
<script src="https://cdn.tailwindcss.com"></script></head>
<body class="bg-gray-50 flex items-center justify-center min-h-screen">
<div class="text-center"><p class="text-7xl font-extrabold text-red-500">403</p>
<h2 class="mt-4 text-xl font-bold text-gray-900">Không có quyền truy cập</h2>
<p class="mt-2 text-sm text-gray-500">Bạn không có quyền truy cập trang này.</p>
<div class="mt-8 flex justify-center gap-3">
<a href="/" class="rounded-xl bg-indigo-600 px-5 py-3 text-sm font-bold text-white hover:bg-indigo-700">Trang chủ</a>
<button onclick="history.back()" class="rounded-xl bg-white border border-gray-200 px-5 py-3 text-sm font-bold text-gray-700 hover:bg-gray-50">Quay lại</button>
</div></div></body></html>`

// Convenience middleware (role-based, no DB lookup)
func AdminOnly(next http.Handler) http.Handler {
	return RequireRoles(models.RoleAdmin)(next)
}

func ManagerOrAdmin(next http.Handler) http.Handler {
	return RequireRoles(models.RoleAdmin, models.RoleManager)(next)
}

// RequirePermission returns middleware that checks if the user's role
// has the specified permission via PermissionService (cached DB lookup).
func RequirePermission(permService *service.PermissionService, permCode string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := GetUserRole(r)
			if !permService.HasPermission(role, permCode) {
				if strings.HasPrefix(r.URL.Path, "/api/") {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte(`{"success":false,"error":{"code":"FORBIDDEN","message":"missing permission: ` + permCode + `"}}`))
					return
				}
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(errorPage403))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyPermission returns middleware that checks if the user's role
// has at least one of the specified permissions.
func RequireAnyPermission(permService *service.PermissionService, permCodes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := GetUserRole(r)
			if !permService.HasAnyPermission(role, permCodes...) {
				if strings.HasPrefix(r.URL.Path, "/api/") {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte(`{"success":false,"error":{"code":"FORBIDDEN","message":"insufficient permissions"}}`))
					return
				}
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(errorPage403))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
