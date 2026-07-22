package middleware

import (
	"net/http"
)

// SecurityHeaders applies standard security headers to prevent common attacks.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent browsers from guessing (sniffing) MIME types, reducing XSS risk.
		w.Header().Set("X-Content-Type-Options", "nosniff")
		
		// Prevent clickjacking attacks by forbidding iframe embedding.
		w.Header().Set("X-Frame-Options", "DENY")

		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		
		// Enable XSS protection filter in browsers.
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		next.ServeHTTP(w, r)
	})
}