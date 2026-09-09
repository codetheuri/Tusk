package tenant_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/codetheuri/tusk/pkg/tenant"
)

// serve runs one request through the middleware and reports what the handler saw.
func serve(t *testing.T, resolve tenant.Resolver) (status int, got uuid.UUID, present, reached bool) {
	t.Helper()

	handler := tenant.Middleware(resolve)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		got, present = tenant.FromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	return rec.Code, got, present, reached
}

func TestMiddleware_PutsTheResolvedTenantInContext(t *testing.T) {
	want := uuid.New()

	status, got, present, reached := serve(t, func(*http.Request) (uuid.UUID, error) {
		return want, nil
	})

	if !reached {
		t.Fatal("handler was not reached")
	}
	if status != http.StatusOK {
		t.Errorf("status %d, want 200", status)
	}
	if !present || got != want {
		t.Errorf("handler saw tenant %v (present=%v), want %v", got, present, want)
	}
}

// TestMiddleware_NoTenantIsNotAnError covers login, registration and health
// checks. They must keep working; a request that goes on to touch a tenanted
// model still fails closed at the query layer.
func TestMiddleware_NoTenantIsNotAnError(t *testing.T) {
	status, _, present, reached := serve(t, func(*http.Request) (uuid.UUID, error) {
		return uuid.Nil, nil
	})

	if !reached {
		t.Fatal("handler was not reached; a request without a tenant must still be served")
	}
	if status != http.StatusOK {
		t.Errorf("status %d, want 200", status)
	}
	if present {
		t.Error("a tenant was placed in context when the resolver reported none")
	}
}

func TestMiddleware_ResolverErrorRejectsTheRequest(t *testing.T) {
	status, _, _, reached := serve(t, func(*http.Request) (uuid.UUID, error) {
		return uuid.New(), errors.New("caller may not act as this tenant")
	})

	if reached {
		t.Error("handler ran despite the resolver refusing the request")
	}
	if status != http.StatusUnauthorized {
		t.Errorf("status %d, want 401", status)
	}
}

// TestMiddleware_IgnoresATenantTheCallerSupplies is the property the whole design
// rests on: the tenant comes from the resolver, and a body or query parameter
// naming a different one changes nothing.
func TestMiddleware_IgnoresATenantTheCallerSupplies(t *testing.T) {
	real, attacker := uuid.New(), uuid.New()

	var seen uuid.UUID
	handler := tenant.Middleware(func(*http.Request) (uuid.UUID, error) {
		return real, nil
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen, _ = tenant.FromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/?tenant_id="+attacker.String(), nil)
	req.Header.Set("X-Tenant-ID", attacker.String())
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if seen != real {
		t.Errorf("handler saw %v, want %v — a caller-supplied value reached the context", seen, real)
	}
}
