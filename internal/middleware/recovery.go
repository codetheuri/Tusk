package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	
	"github.com/codetheuri/tusk/pkg/logger"
	
)

// recover from panics and return a 500 error
func Recovery(log logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rcvErr := recover(); rcvErr != nil {
					//log thr panic
					var actualErr error
					if e,  ok := rcvErr.(error); ok {
						actualErr = e
					} else {
						actualErr = fmt.Errorf("%v", rcvErr)
					}
					log.Error("PANIC_RECOVERED", actualErr, "stack_trace", string(debug.Stack()))
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"success":false,"message":"Internal server error"}`))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
