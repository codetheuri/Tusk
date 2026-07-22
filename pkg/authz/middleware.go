package authz

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"gorm.io/gorm"

	"github.com/codetheuri/tusk/pkg/response"
)

// Security Requirement Helpers for Huma OpenAPI documentation.
var (
	// BearerSecurity marks an operation as requiring JWT Bearer authentication.
	BearerSecurity = []map[string][]string{{"bearerAuth": {}}}

	// PublicSecurity explicitly marks an operation as unauthenticated/public.
	PublicSecurity = []map[string][]string{}
)

// SubjectExtractor defines a function signature for extracting Subject from context.Context.
type SubjectExtractor func(ctx context.Context) (Subject, bool)

// DefaultSubjectExtractor extracts UserID and IsSuperUser from context standard keys.
var DefaultSubjectExtractor SubjectExtractor = func(ctx context.Context) (Subject, bool) {
	var sub Subject

	// Extract UserID
	if uid, ok := ctx.Value("user_id").(uint); ok {
		sub.UserID = uid
	} else if uidInt, ok := ctx.Value("user_id").(int); ok {
		sub.UserID = uint(uidInt)
	} else {
		return sub, false
	}

	// Extract IsSuperUser flag
	if isSuper, ok := ctx.Value("is_super_user").(bool); ok {
		sub.IsSuperUser = isSuper
	}

	return sub, true
}

// RequirePolicy creates a HTTP middleware that enforces any Policy rule.
func RequirePolicy(db *gorm.DB, policy Policy) func(http.Handler) http.Handler {
	evaluator := NewEvaluator(db)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sub, ok := DefaultSubjectExtractor(r.Context())
			if !ok {
				response.WriteJSON(w, http.StatusUnauthorized, `{"success":false,"message":"Unauthorized: authentication required","errors":null}`)
				return
			}

			authorized, err := evaluator.IsAuthorized(r.Context(), sub, policy)
			if err != nil {
				response.WriteJSON(w, http.StatusInternalServerError, `{"success":false,"message":"Internal authorization error","errors":null}`)
				return
			}

			if !authorized {
				response.WriteJSON(w, http.StatusForbidden, `{"success":false,"message":"Forbidden: insufficient permissions","errors":null}`)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission demands a single specific permission constant.
func RequirePermission(db *gorm.DB, permission string) func(http.Handler) http.Handler {
	return RequirePolicy(db, RequirePermissionPolicy{Permission: permission})
}

// RequireAny demands that the user has at least one of the specified permissions.
func RequireAny(db *gorm.DB, permissions ...string) func(http.Handler) http.Handler {
	return RequirePolicy(db, RequireAnyPolicy{Permissions: permissions})
}

// RequireAll demands that the user possesses ALL of the specified permissions.
func RequireAll(db *gorm.DB, permissions ...string) func(http.Handler) http.Handler {
	return RequirePolicy(db, RequireAllPolicy{Permissions: permissions})
}

// --- Huma Guard & Middleware Hooks ---

// Guard holds references to huma.API and Evaluator for clean route protection.
type Guard struct {
	api       huma.API
	evaluator *Evaluator
}

// NewGuard constructs a reusable Guard for route authorization.
func NewGuard(api huma.API, db *gorm.DB) *Guard {
	return &Guard{
		api:       api,
		evaluator: NewEvaluator(db),
	}
}

// Protected decorates a Huma operation with BearerSecurity and a required permission.
func (g *Guard) Protected(op huma.Operation, permission string) huma.Operation {
	op.Security = BearerSecurity
	if permission != "" {
		if op.Middlewares == nil {
			op.Middlewares = huma.Middlewares{}
		}
		op.Middlewares = append(op.Middlewares, g.Require(permission))
	}
	return op
}

// Require returns a clean Huma middleware hook demanding a specific permission.
func (g *Guard) Require(permission string) func(huma.Context, func(huma.Context)) {
	policy := RequirePermissionPolicy{Permission: permission}
	return func(ctx huma.Context, next func(huma.Context)) {
		sub, ok := DefaultSubjectExtractor(ctx.Context())
		if !ok {
			huma.WriteErr(g.api, ctx, http.StatusUnauthorized, "Unauthorized: authentication required")
			return
		}

		authorized, err := g.evaluator.IsAuthorized(ctx.Context(), sub, policy)
		if err != nil || !authorized {
			huma.WriteErr(g.api, ctx, http.StatusForbidden, "Forbidden: insufficient permissions")
			return
		}

		next(ctx)
	}
}

// RequireAny returns a clean Huma middleware hook demanding at least one of the specified permissions.
func (g *Guard) RequireAny(permissions ...string) func(huma.Context, func(huma.Context)) {
	policy := RequireAnyPolicy{Permissions: permissions}
	return func(ctx huma.Context, next func(huma.Context)) {
		sub, ok := DefaultSubjectExtractor(ctx.Context())
		if !ok {
			huma.WriteErr(g.api, ctx, http.StatusUnauthorized, "Unauthorized: authentication required")
			return
		}

		authorized, err := g.evaluator.IsAuthorized(ctx.Context(), sub, policy)
		if err != nil || !authorized {
			huma.WriteErr(g.api, ctx, http.StatusForbidden, "Forbidden: insufficient permissions")
			return
		}

		next(ctx)
	}
}

// HumaRequirePermission returns a Huma middleware hook for clean Huma integration.
func HumaRequirePermission(api huma.API, db *gorm.DB, permission string) func(huma.Context, func(huma.Context)) {
	return NewGuard(api, db).Require(permission)
}
