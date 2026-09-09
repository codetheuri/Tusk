// Package tenant provides optional multi-tenancy: many customers sharing one
// application and one database, with each customer's rows invisible to the others.
//
// # Tenancy is opt-in, per model
//
// Most applications are single-tenant, and a framework that stamped a tenant
// column onto every table would be unusable for them. Nothing here activates
// unless a model asks for it by implementing Tenanted:
//
//	func (Customer) TenantColumn() string { return "business_id" }
//
// A model without that method is untouched — no column, no injected predicate,
// no runtime cost. An application that declares no tenant-scoped models behaves
// exactly as if this package did not exist.
//
// Returning the column *name* rather than assuming "tenant_id" lets each
// application keep its own vocabulary: a shop system says business_id, a
// workspace product says org_id.
//
// # Three layers, and why one is not enough
//
//  1. Context — the tenant is resolved once, at the edge, from a trusted source
//     such as a JWT claim. Never from a request body, which the caller controls.
//  2. Query scoping — a GORM callback adds the tenant predicate to every read,
//     update and delete of a tenanted model. This is the layer that matters:
//     hand-written filters get forgotten, and one forgotten filter is a breach.
//  3. Row-level security — PostgreSQL policies refuse foreign rows regardless of
//     what the application asks for. A backstop for the day layer 2 has a bug.
//
// Layer 2 fails closed: a tenanted model queried with no tenant in context
// returns an error rather than every row in the table. Legitimate exceptions —
// migrations, admin tooling, background jobs — must say so explicitly with
// Unscoped, which is greppable in review.
package tenant

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// Tenanted marks a model as belonging to a tenant.
//
// The single method is the entire opt-in mechanism. Implement it and every query
// against the model is scoped automatically; omit it and nothing changes.
type Tenanted interface {
	// TenantColumn returns the column holding the tenant identifier,
	// for example "business_id" or "org_id".
	TenantColumn() string
}

// ErrNoTenant is returned when a tenanted model is accessed with no tenant in
// context and no explicit bypass.
//
// This is deliberately an error and not a silent unscoped query. The failure
// mode of guessing is that one customer sees another's data; the failure mode of
// erroring is a broken request, which is noisy, obvious, and safe.
var ErrNoTenant = errors.New("tenant: no tenant in context (use tenant.Unscoped for deliberate cross-tenant access)")

// tenantKey is an unexported context key type, so no other package can collide
// with it — including one that uses the same string.
type tenantKey struct{}

// WithTenant returns a context carrying the tenant identifier.
//
// Call this once per request, from a trusted source. Resolving the tenant from
// anything the caller supplies — a body field, a query parameter, a header they
// control — would let one tenant simply ask for another's data.
func WithTenant(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, tenantKey{}, id)
}

// FromContext returns the tenant identifier and whether one is present.
//
// The boolean must be checked: the zero UUID is a valid-looking value and would
// silently scope queries to a tenant that does not exist.
func FromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(tenantKey{}).(uuid.UUID)
	if !ok || id == uuid.Nil {
		return uuid.Nil, false
	}
	return id, true
}

// MustFromContext returns the tenant identifier, or an error if absent.
// Useful in services that cannot meaningfully proceed without one.
func MustFromContext(ctx context.Context) (uuid.UUID, error) {
	id, ok := FromContext(ctx)
	if !ok {
		return uuid.Nil, ErrNoTenant
	}
	return id, nil
}
