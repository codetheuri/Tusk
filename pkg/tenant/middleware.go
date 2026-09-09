package tenant

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"github.com/codetheuri/tusk/v2/pkg/response"
)

// Resolver determines the active tenant for a request.
//
// Two outcomes are not failures and must be distinguished:
//
//   - (uuid.Nil, nil) — no tenant applies. The request continues unscoped, and
//     any tenanted query it goes on to make fails closed with ErrNoTenant. This
//     is the correct answer for login, registration and health checks.
//   - (id, nil) — this request acts as that tenant.
//
// An error rejects the request outright, for a credential that names a tenant
// the caller may not act as.
//
// A Resolver must read only from sources the caller cannot choose: a verified
// token claim, a subdomain matched against a lookup, a gateway header on a
// trusted network. Never a request body or query parameter — a tenant taken from
// those is not an identity, it is a request to be someone else.
type Resolver func(r *http.Request) (uuid.UUID, error)

// HumaResolver is Resolver for the Huma API surface.
type HumaResolver func(ctx huma.Context) (uuid.UUID, error)

// Middleware puts the resolved tenant into the request context.
//
// It is the only supported way for a tenant to enter the system. Handlers read
// it from the context and never accept it as a parameter, so no route can be
// written that takes the caller's word for who they are.
//
// Nothing here activates on its own: an application that registers no middleware
// simply never has a tenant in context, and models that never opted in are
// unaffected either way.
func Middleware(resolve Resolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, err := resolve(r)
			if err != nil {
				response.WriteJSON(w, http.StatusUnauthorized,
					`{"success":false,"message":"Unable to determine the tenant for this request"}`)
				return
			}
			if id == uuid.Nil {
				next.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithTenant(r.Context(), id)))
		})
	}
}

// HumaMiddleware is Middleware for the Huma API surface.
func HumaMiddleware(api huma.API, resolve HumaResolver) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		id, err := resolve(ctx)
		if err != nil {
			huma.WriteErr(api, ctx, http.StatusUnauthorized,
				"Unable to determine the tenant for this request")
			return
		}
		if id == uuid.Nil {
			next(ctx)
			return
		}
		next(huma.WithContext(ctx, WithTenant(ctx.Context(), id)))
	}
}
