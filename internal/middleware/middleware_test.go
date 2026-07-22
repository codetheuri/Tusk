package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type dummyLogger struct {
	buf bytes.Buffer
}

func newDummyLogger() *dummyLogger {
	return &dummyLogger{}
}

func (d *dummyLogger) Debug(msg string, args ...any) {}
func (d *dummyLogger) Info(msg string, args ...any)  {}
func (d *dummyLogger) Warn(msg string, args ...any)  {}
func (d *dummyLogger) Error(msg string, err error, args ...any) {
	d.buf.WriteString(msg)
}
func (d *dummyLogger) Fatal(msg string, err error, args ...any) {
	log.Fatal(msg)
}

func TestRequestIDMiddleware(t *testing.T) {
	handler := RequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id == "" {
			t.Error("expected non-empty request id in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header in response")
	}
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("expected X-Content-Type-Options nosniff")
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("expected X-Frame-Options DENY")
	}
}

func TestCORSMiddleware(t *testing.T) {
	dl := newDummyLogger()
	handler := CORS([]string{"https://example.com"}, dl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// OPTIONS preflight
	req := httptest.NewRequest("OPTIONS", "/api/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK for OPTIONS preflight, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("expected Access-Control-Allow-Origin header")
	}
	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials header to be true")
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	dl := newDummyLogger()
	handler := Recovery(dl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went terribly wrong")
	}))

	req := httptest.NewRequest("GET", "/panic", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 status code on panic, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected application/json content type on panic recovery")
	}
}

func TestJWTMiddleware_Authenticate(t *testing.T) {
	secret := "super-secret-key"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		UserID: 1,
		Role:   "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	tokenStr, _ := token.SignedString([]byte(secret))

	handler := Authenticate(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid := GetUserID(r.Context())
		if uid != 1 {
			t.Errorf("expected user_id 1 in context, got %d", uid)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK for valid JWT, got %d", rec.Code)
	}
}
