package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// A handler behind the logger must still be able to extend its own write
// deadline: http.ResponseController finds the connection through Unwrap.
func TestLoggerLetsHandlersReachTheConnection(t *testing.T) {
	var deadlineErr, flushErr error
	handler := Logger(newDummyLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rc := http.NewResponseController(w)
		deadlineErr = rc.SetWriteDeadline(time.Now().Add(time.Hour))
		_, _ = io.WriteString(w, "ok")
		flushErr = rc.Flush()
	}))

	srv := httptest.NewServer(handler)
	defer srv.Close()
	res, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	if deadlineErr != nil {
		t.Errorf("SetWriteDeadline through the logger: %v", deadlineErr)
	}
	if flushErr != nil {
		t.Errorf("Flush through the logger: %v", flushErr)
	}
}

func TestLoggerRecordsTheStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	Logger(newDummyLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusTeapot {
		t.Fatalf("got %d", rec.Code)
	}
}
