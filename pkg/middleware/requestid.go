package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// contextKey is a private named type for context keys owned by this package.
//
// The distinction that matters: a *named* type like this is safe, because
// context lookups compare the key's type as well as its value — no other package
// can construct a middleware.contextKey. An *untyped* string literal such as
// ctx.Value("user_id") is not safe, because any package writing that same literal
// collides silently. Both look similar at the call site; only one of them works.
type contextKey string

const (
	RequestIDKey contextKey = "requestID"
)

// RequestID generates a unique request ID
func RequestID() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			id := uuid.New().String()

			ctx := context.WithValue(r.Context(), RequestIDKey, id)
			r = r.WithContext(ctx)

			w.Header().Set("X-Request-ID", id)

			next.ServeHTTP(w, r)
		})
	}
}

func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}
