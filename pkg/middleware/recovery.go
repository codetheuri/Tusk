package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	appErrors "github.com/codetheuri/todolist/pkg/errors"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/codetheuri/todolist/pkg/web"
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
					log.Error("PANIC_RECOVERED", actualErr, "stack_trace", string(debug.Stack()),)
					web.RespondError(w, appErrors.New("INTERNAL_SERVER_ERROR", "An unexpected error occurred", nil), http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})

	}
}
