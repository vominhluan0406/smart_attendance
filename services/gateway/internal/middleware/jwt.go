package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/smart-attendance/shared/response"
)

// Claims matches the JWT claims structure used by auth-service.
type Claims struct {
	jwt.RegisteredClaims
	UserID   string  `json:"user_id"`
	Email    string  `json:"email"`
	FullName string  `json:"full_name"`
	Role     string  `json:"role"`
	BranchID *string `json:"branch_id,omitempty"`
}

// JWTAuth returns middleware that validates JWT tokens locally using the shared secret.
// On success, it injects X-User-ID, X-User-Role, X-User-Email, X-Branch-ID headers
// into the request for downstream services.
func JWTAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString := extractToken(r)
			if tokenString == "" {
				log.Printf("[gateway][jwt] missing token for %s %s", r.Method, r.URL.Path)
				response.Error(w, http.StatusUnauthorized, "MISSING_TOKEN",
					"authorization token is required")
				return
			}

			claims, err := validateToken(tokenString, jwtSecret)
			if err != nil {
				log.Printf("[gateway][jwt] invalid token for %s %s: %v", r.Method, r.URL.Path, err)
				response.Error(w, http.StatusUnauthorized, "INVALID_TOKEN",
					"invalid or expired token")
				return
			}

			// Inject identity headers for downstream services
			r.Header.Set("X-User-ID", claims.UserID)
			r.Header.Set("X-User-Role", claims.Role)
			r.Header.Set("X-User-Email", claims.Email)
			if claims.BranchID != nil {
				r.Header.Set("X-Branch-ID", *claims.BranchID)
			}

			// Forward the original Authorization header so downstream services
			// can also validate if needed
			r.Header.Set("Authorization", "Bearer "+tokenString)

			next.ServeHTTP(w, r)
		})
	}
}

// extractToken gets the JWT token from the Authorization header or access_token cookie.
func extractToken(r *http.Request) string {
	// Try Authorization header first
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != authHeader {
			return token
		}
	}

	// Fallback to cookie
	cookie, err := r.Cookie("access_token")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	return ""
}

// validateToken parses and validates a JWT token using the shared secret.
func validateToken(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("token parse error: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	if claims.UserID == "" {
		return nil, fmt.Errorf("token missing user_id claim")
	}

	return claims, nil
}
