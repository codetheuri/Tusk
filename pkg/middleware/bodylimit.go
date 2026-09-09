package middleware

import "net/http"

// MaxBodyBytes caps how much of a request body the server will read.
//
// Without a cap, a single client can stream an unbounded body and the process
// will keep allocating until it is killed — no exploit required, just a slow
// upload that never ends. http.MaxBytesReader stops the read at the limit and
// closes the connection, so the cost of an oversized request is bounded.
//
// The limit applies to the body only. Header size is capped separately by
// http.Server.MaxHeaderBytes.
func MaxBodyBytes(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}
			next.ServeHTTP(w, r)
		})
	}
}
