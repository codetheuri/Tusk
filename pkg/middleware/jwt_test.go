package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/codetheuri/tusk/v2/pkg/authz"
)

const testSecret = "test-secret-key-that-is-long-enough-for-hs256"

// signToken produces a token the middleware should accept.
func signToken(t *testing.T, claims Claims) string {
	t.Helper()
	if claims.ExpiresAt == nil {
		claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(time.Hour))
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("failed to sign test token: %v", err)
	}
	return signed
}

func TestParseBearerToken(t *testing.T) {
	valid := signToken(t, Claims{UserID: testUserID(7), IsSuperUser: true})
	expired := signToken(t, Claims{
		UserID:           testUserID(7),
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour))},
	})

	// A token signed with a different key must be rejected even though it is
	// structurally valid — this is the signature check doing its job.
	forged, err := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID:           testUserID(99),
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	}).SignedString([]byte("a-completely-different-signing-key-value"))
	if err != nil {
		t.Fatalf("failed to sign forged token: %v", err)
	}

	tests := []struct {
		name       string
		header     string
		wantErr    bool
		wantNoCred bool
		wantUserID uuid.UUID
	}{
		{name: "valid bearer token", header: "Bearer " + valid, wantUserID: testUserID(7)},
		{name: "case-insensitive scheme", header: "bearer " + valid, wantUserID: testUserID(7)},
		{name: "empty header reports no credentials", header: "", wantErr: true, wantNoCred: true},
		{name: "missing scheme", header: valid, wantErr: true},
		{name: "wrong scheme", header: "Basic " + valid, wantErr: true},
		{name: "expired token", header: "Bearer " + expired, wantErr: true},
		{name: "token signed with another key", header: "Bearer " + forged, wantErr: true},
		{name: "garbage token", header: "Bearer not-a-jwt", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			claims, err := parseBearerToken(tc.header, testSecret)

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				// The distinction between "no credentials" and "bad credentials"
				// is what lets the Huma middleware answer 401 instead of falling
				// through to a misleading 403.
				if got := errors.Is(err, ErrNoCredentials); got != tc.wantNoCred {
					t.Errorf("errors.Is(err, ErrNoCredentials) = %v, want %v (err: %v)", got, tc.wantNoCred, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if claims.UserID != tc.wantUserID {
				t.Errorf("UserID = %s, want %s", claims.UserID, tc.wantUserID)
			}
		})
	}
}

// IsSuperUser now travels in the token instead of being fetched per request.
func TestParseBearerToken_CarriesSuperUserFlag(t *testing.T) {
	super := signToken(t, Claims{UserID: testUserID(1), IsSuperUser: true})
	ordinary := signToken(t, Claims{UserID: testUserID(2)})

	claims, err := parseBearerToken("Bearer "+super, testSecret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !claims.IsSuperUser {
		t.Error("expected IsSuperUser true for a super-user token")
	}

	claims, err = parseBearerToken("Bearer "+ordinary, testSecret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.IsSuperUser {
		t.Error("expected IsSuperUser false when the claim is absent — it must fail closed")
	}
}

func TestAuthenticate_RejectsAnonymous(t *testing.T) {
	handlerRan := false
	h := Authenticate(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerRan = true
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rr.Code)
	}
	if handlerRan {
		t.Error("handler ran despite missing credentials")
	}
}

func TestAuthenticate_PopulatesSubject(t *testing.T) {
	var got authz.Subject
	var ok bool

	h := Authenticate(testSecret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, ok = authz.SubjectFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+signToken(t, Claims{UserID: testUserID(42), IsSuperUser: true}))
	h.ServeHTTP(httptest.NewRecorder(), req)

	if !ok {
		t.Fatal("expected a Subject in context")
	}
	if got.UserID != testUserID(42) || !got.IsSuperUser {
		t.Errorf("Subject = %+v, want {UserID:%s IsSuperUser:true}", got, testUserID(42))
	}
}

// This is the H3 regression guard. An absent credential is legitimate on a public
// route; a malformed one never is, and conflating them told clients "forbidden"
// when the truth was "your session expired".
func TestHumaAuthenticate_DistinguishesAbsentFromInvalid(t *testing.T) {
	t.Run("no token proceeds anonymously", func(t *testing.T) {
		reached := false
		api := newProbeAPI(t, func() { reached = true })

		resp := api.Get("/probe")
		if resp.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", resp.Code)
		}
		if !reached {
			t.Error("expected the handler to run for an anonymous request")
		}
	})

	t.Run("invalid token is refused with 401", func(t *testing.T) {
		reached := false
		api := newProbeAPI(t, func() { reached = true })

		resp := api.Get("/probe", "Authorization: Bearer not-a-valid-token")
		if resp.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", resp.Code)
		}
		if reached {
			t.Error("handler ran despite an invalid token")
		}
	})

	t.Run("expired token is refused with 401", func(t *testing.T) {
		api := newProbeAPI(t, func() {})

		expired := signToken(t, Claims{
			UserID:           testUserID(1),
			RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute))},
		})

		resp := api.Get("/probe", "Authorization: Bearer "+expired)
		if resp.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401 (an expired session must be distinguishable from a permission failure)", resp.Code)
		}
	})
}

// probeOutput is the response of the throwaway endpoint used to observe whether
// a request reached its handler.
type probeOutput struct {
	Body struct {
		OK bool `json:"ok"`
	}
}

// newProbeAPI builds a Huma API with the authentication middleware installed and
// a single unguarded endpoint, so a test can distinguish "refused at the edge"
// from "reached the handler".
func newProbeAPI(t *testing.T, onReach func()) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	api.UseMiddleware(HumaAuthenticate(api, testSecret))
	huma.Get(api, "/probe", func(ctx context.Context, _ *struct{}) (*probeOutput, error) {
		onReach()
		return &probeOutput{}, nil
	})
	return api
}

// testUserID builds a deterministic identifier from a small integer, so tests can
// keep using readable IDs instead of hard-coded UUID literals. The value only has
// to be stable and distinct — it never reaches a database.
func testUserID(n int) uuid.UUID {
	var u uuid.UUID
	u[15] = byte(n)
	return u
}
