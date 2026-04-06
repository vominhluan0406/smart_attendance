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
	window   time.Duration
}

var (
	ipLimiter = &rateLimiter{
		requests: make(map[string][]time.Time),
		window:   time.Minute,
	}
	userLimiter = &rateLimiter{
		requests: make(map[string][]time.Time),
		window:   time.Minute,
	}
)

func rateLimitByKey(rl *rateLimiter, key string, max int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	entries := rl.requests[key]
	var valid []time.Time
	for _, t := range entries {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= max {
		rl.requests[key] = valid
		return false // Rate limited
	}

	valid = append(valid, now)
	rl.requests[key] = valid
	return true // Allowed
}

func writeRateLimitError(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.Error(w, `{"success":false,"error":{"code":"RATE_LIMITED","message":"too many requests"}}`, http.StatusTooManyRequests)
	} else {
		http.Error(w, "Quá nhiều yêu cầu. Vui lòng thử lại sau.", http.StatusTooManyRequests)
	}
}

// RateLimit limits requests per IP per minute.
func RateLimit(maxPerMinute int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r)
			if !rateLimitByKey(ipLimiter, ip, maxPerMinute) {
				writeRateLimitError(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitByUser limits requests per user_id (from JWT context) per minute.
// This prevents a single user from spamming check-in attempts even on shared WiFi.
func RateLimitByUser(maxPerMin int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := GetUserID(r)
			if userID == "" {
				next.ServeHTTP(w, r)
				return
			}
			key := "user:" + userID
			if !rateLimitByKey(userLimiter, key, maxPerMin) {
				writeRateLimitError(w, r)
				return
			}
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
