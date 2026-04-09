package middleware

import (
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/smart-attendance/shared/response"
	"golang.org/x/time/rate"
)

// visitor holds the rate limiter and last seen time for a client IP.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter provides IP-based rate limiting middleware.
type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.Mutex
	limit    rate.Limit
	burst    int
}

// NewRateLimiter creates a new rate limiter with the given per-minute limit.
func NewRateLimiter(perMinute int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    rate.Limit(float64(perMinute) / 60.0), // convert per-minute to per-second
		burst:    perMinute,                              // allow bursts up to the full per-minute limit
	}

	// Start cleanup goroutine to remove stale entries
	go rl.cleanup()

	return rl
}

// getVisitor returns the rate limiter for the given IP, creating one if needed.
func (rl *RateLimiter) getVisitor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(rl.limit, rl.burst)
		rl.visitors[ip] = &visitor{limiter: limiter, lastSeen: time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

// cleanup removes visitors that haven't been seen for 3 minutes.
func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(1 * time.Minute)
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Handler returns the rate limiting middleware.
func (rl *RateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract client IP
		ip := extractClientIP(r)

		limiter := rl.getVisitor(ip)
		if !limiter.Allow() {
			log.Printf("[gateway][ratelimit] rate limit exceeded for IP %s on %s %s", ip, r.Method, r.URL.Path)
			w.Header().Set("Retry-After", "60")
			response.Error(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED",
				"too many requests, please try again later")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// extractClientIP gets the real client IP from headers or RemoteAddr.
func extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For first (set by load balancers/proxies)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// Take the first IP in the chain (original client)
		ip := forwarded
		if idx := len(forwarded); idx > 0 {
			parts := splitFirst(forwarded, ",")
			ip = parts
		}
		return ip
	}

	// Check X-Real-IP
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// splitFirst returns the substring before the first occurrence of sep.
func splitFirst(s, sep string) string {
	for i := 0; i < len(s); i++ {
		if string(s[i]) == sep {
			return s[:i]
		}
	}
	return s
}
