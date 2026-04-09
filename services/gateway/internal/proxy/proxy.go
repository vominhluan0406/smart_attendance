package proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/smart-attendance/shared/response"
)

// NewServiceProxy creates a reverse proxy that forwards requests to the target service.
// It strips the matched route prefix and forwards the remaining path.
func NewServiceProxy(targetURL string) http.Handler {
	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("[gateway][proxy] invalid target URL %s: %v", targetURL, err)
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host

			// Preserve the original path (chi handles prefix stripping via route patterns)
			// The request path at this point is already the full matched path
			if _, ok := req.Header["User-Agent"]; !ok {
				req.Header.Set("User-Agent", "")
			}

			// Forward client IP
			if clientIP := req.RemoteAddr; clientIP != "" {
				prior := req.Header.Get("X-Forwarded-For")
				if prior != "" {
					req.Header.Set("X-Forwarded-For", prior+", "+clientIP)
				} else {
					req.Header.Set("X-Forwarded-For", clientIP)
				}
			}

			log.Printf("[gateway][proxy] forwarding %s %s -> %s%s",
				req.Method, req.URL.Path, target.Host, req.URL.Path)
		},
		ModifyResponse: func(resp *http.Response) error {
			// Copy hop-by-hop headers are handled automatically by ReverseProxy
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("ERROR: [gateway][proxy] upstream error for %s %s: %v", r.Method, r.URL.Path, err)
			response.Error(w, http.StatusBadGateway, "BAD_GATEWAY",
				"service unavailable: "+err.Error())
		},
	}

	return proxy
}

// NewPrefixProxy creates a reverse proxy that strips a URL prefix before forwarding.
// For example, if prefix is "/api/attendance" and request is "/api/attendance/check-in",
// the forwarded request will be "/api/attendance/check-in" to the target.
func NewPrefixProxy(targetURL, prefix string) http.Handler {
	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("[gateway][proxy] invalid target URL %s: %v", targetURL, err)
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host

			// Keep the original path as-is — downstream services define the same routes
			originalPath := req.URL.Path
			if !strings.HasPrefix(originalPath, prefix) {
				// Fallback: just forward as-is
				log.Printf("[gateway][proxy] path %s does not start with prefix %s, forwarding as-is", originalPath, prefix)
			}

			if _, ok := req.Header["User-Agent"]; !ok {
				req.Header.Set("User-Agent", "")
			}

			// Forward client IP
			if clientIP := req.RemoteAddr; clientIP != "" {
				prior := req.Header.Get("X-Forwarded-For")
				if prior != "" {
					req.Header.Set("X-Forwarded-For", prior+", "+clientIP)
				} else {
					req.Header.Set("X-Forwarded-For", clientIP)
				}
			}

			log.Printf("[gateway][proxy] forwarding %s %s -> %s%s",
				req.Method, req.URL.Path, target.Host, req.URL.Path)
		},
		ModifyResponse: func(resp *http.Response) error {
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("ERROR: [gateway][proxy] upstream error for %s %s: %v", r.Method, r.URL.Path, err)
			response.Error(w, http.StatusBadGateway, "BAD_GATEWAY",
				"service unavailable: "+err.Error())
		},
	}

	return proxy
}
