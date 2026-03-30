package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

type rateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

var limiter = &rateLimiter{
	requests: make(map[string][]time.Time),
	limit:    10,
	window:   time.Minute,
}

// RateLimit limits requests per IP per minute.
func RateLimit(maxPerMinute int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r)

			limiter.mu.Lock()
			now := time.Now()
			cutoff := now.Add(-limiter.window)

			// Clean old entries
			entries := limiter.requests[ip]
			var valid []time.Time
			for _, t := range entries {
				if t.After(cutoff) {
					valid = append(valid, t)
				}
			}

			if len(valid) >= maxPerMinute {
				limiter.mu.Unlock()
				if strings.HasPrefix(r.URL.Path, "/api/") {
					http.Error(w, `{"success":false,"error":{"code":"RATE_LIMITED","message":"too many requests"}}`, http.StatusTooManyRequests)
				} else {
					http.Error(w, "Too many requests. Please try again later.", http.StatusTooManyRequests)
				}
				return
			}

			valid = append(valid, now)
			limiter.requests[ip] = valid
			limiter.mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}

func extractIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-Ip"); ip != "" {
		return ip
	}
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.Split(forwarded, ",")[0]
	}
	return r.RemoteAddr
}
