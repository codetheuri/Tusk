package middleware

import (
	"net/http"

	"github.com/codetheuri/tusk/pkg/logger"
)

func CORS(allowedOrigins []string, log logger.Logger) func(next http.Handler) http.Handler {
	if len(allowedOrigins) == 0 {
		log.Warn("CORS middleware initialized with no allowed origins")
		allowedOrigins = []string{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			isAllowed := false

			// Check if the request origin is in the allowed list or if "*" is allowed.
			for _, allowed := range allowedOrigins {
				if allowed == "*" || allowed == origin {
					isAllowed = true
					break
				}
			}

			if isAllowed {
				if origin != "" {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				} else {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				}
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, X-Request-ID")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}