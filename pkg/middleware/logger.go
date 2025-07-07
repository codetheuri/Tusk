package middleware

import (
	"net/http"
	"time"

	"github.com/codetheuri/todolist/pkg/logger"
)

func Logger(log logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			start := time.Now()

			//custom response writer to capture status code
			lrw := newLoggingResponseWriter(w)

			next.ServeHTTP(w, r)

			log.Info("HTTP Request",
				"method", r.Method,
				"url", r.RequestURI,
				"status", lrw.statusCode,
				"duration", time.Since(start),
				"client_ip", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			)
		})
	}
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}
func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter{
	return &loggingResponseWriter{w, http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}