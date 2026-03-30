package middleware

import (
	"log"
	"net/http"
	"runtime/debug"
	"strings"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC: %v\n%s", err, debug.Stack())
				if strings.HasPrefix(r.URL.Path, "/api/") {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"success":false,"error":{"code":"INTERNAL_ERROR","message":"internal server error"}}`))
					return
				}
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(errorPage500))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Minimal fallback HTML for panic recovery (renderer may not be available).
const errorPage500 = `<!DOCTYPE html>
<html lang="vi"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>500 — Smart Attendance</title>
<script src="https://cdn.tailwindcss.com"></script></head>
<body class="bg-gray-50 flex items-center justify-center min-h-screen">
<div class="text-center"><p class="text-7xl font-extrabold text-gray-400">500</p>
<h2 class="mt-4 text-xl font-bold text-gray-900">Lỗi hệ thống</h2>
<p class="mt-2 text-sm text-gray-500">Đã xảy ra lỗi không mong muốn. Vui lòng thử lại sau.</p>
<a href="/" class="mt-8 inline-block rounded-xl bg-indigo-600 px-5 py-3 text-sm font-bold text-white hover:bg-indigo-700">Trang chủ</a>
</div></body></html>`
