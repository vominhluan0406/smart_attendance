package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/service"
)

type contextKey string

const (
	ContextUserID   contextKey = "user_id"
	ContextEmail    contextKey = "email"
	ContextRole     contextKey = "role"
	ContextBranchID contextKey = "branch_id"
)

func JWTAuth(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				// Check if HTMX request — redirect to login
				if r.Header.Get("HX-Request") == "true" {
					w.Header().Set("HX-Redirect", "/auth/login")
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				// Check if API request
				if strings.HasPrefix(r.URL.Path, "/api/") {
					http.Error(w, `{"success":false,"error":{"code":"UNAUTHORIZED","message":"missing or invalid token"}}`, http.StatusUnauthorized)
					return
				}
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}

			claims, err := authService.ValidateToken(token)
			if err != nil {
				if r.Header.Get("HX-Request") == "true" {
					w.Header().Set("HX-Redirect", "/auth/login")
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				if strings.HasPrefix(r.URL.Path, "/api/") {
					http.Error(w, `{"success":false,"error":{"code":"UNAUTHORIZED","message":"invalid or expired token"}}`, http.StatusUnauthorized)
					return
				}
				http.Redirect(w, r, "/auth/login", http.StatusFound)
				return
			}

			ctx := context.WithValue(r.Context(), ContextUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextEmail, claims.Email)
			ctx = context.WithValue(ctx, ContextRole, claims.Role)
			if claims.BranchID != nil {
				ctx = context.WithValue(ctx, ContextBranchID, *claims.BranchID)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractToken(r *http.Request) string {
	// 1. Authorization header
	if auth := r.Header.Get("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
	}
	// 2. Cookie
	if cookie, err := r.Cookie("access_token"); err == nil {
		return cookie.Value
	}
	return ""
}

// Helper functions to extract values from context
func GetUserID(r *http.Request) string {
	if val, ok := r.Context().Value(ContextUserID).(string); ok {
		return val
	}
	return ""
}

func GetUserRole(r *http.Request) models.Role {
	if val, ok := r.Context().Value(ContextRole).(models.Role); ok {
		return val
	}
	return ""
}

func GetBranchID(r *http.Request) string {
	if val, ok := r.Context().Value(ContextBranchID).(string); ok {
		return val
	}
	return ""
}

func GetUserEmail(r *http.Request) string {
	if val, ok := r.Context().Value(ContextEmail).(string); ok {
		return val
	}
	return ""
}
