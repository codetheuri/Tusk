// Package middleware provides the global HTTP middleware chain: request IDs,
// logging, panic recovery, CORS, security headers, rate limiting, and
// authentication.
package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/codetheuri/tusk/pkg/authz"
	"github.com/codetheuri/tusk/pkg/response"
	"github.com/codetheuri/tusk/pkg/tenant"
)

// Claims is the payload carried by a Tusk access token.
//
// IsSuperUser is embedded in the token rather than looked up per request. The
// tradeoff is deliberate: a revoked super-user retains the flag until their
// token expires. That is acceptable for a rarely-changing administrative flag
// and it removes a database round-trip from every authenticated request. Ordinary
// permissions are NOT stored here for exactly the opposite reason — they change
// often, so they are evaluated against the database on each check by pkg/authz.
//
// Role is retained for applications that want a coarse role claim. Tusk's own
// authorization does not consult it; use pkg/authz permissions instead.
//
// TenantID is omitempty and unused by single-tenant applications, which never
// set it and never read it. It belongs in the token rather than in a lookup for
// the same reason the tenant must not come from the request body: it has to be
// something the caller cannot choose.
type Claims struct {
	UserID      uuid.UUID `json:"user_id"`
	TenantID    uuid.UUID `json:"tenant_id,omitempty"`
	Role        string    `json:"role,omitempty"`
	IsSuperUser bool      `json:"is_super_user,omitempty"`
	jwt.RegisteredClaims
}

// ErrNoCredentials indicates the request carried no Authorization header at all.
// It is distinct from a parse or validation failure: an absent credential may be
// legitimate on a public route, whereas a malformed one never is.
var ErrNoCredentials = errors.New("no credentials presented")

// parseBearerToken extracts and verifies a JWT from an Authorization header.
//
// This is the single place tokens are parsed. Both the net/http and the Huma
// middleware delegate here so the two paths cannot drift apart in how they
// validate signatures or interpret claims.
func parseBearerToken(authHeader, jwtSecret string) (*Claims, error) {
	if authHeader == "" {
		return nil, ErrNoCredentials
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return nil, fmt.Errorf("invalid authorization format, expected: Bearer <token>")
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (any, error) {
		// Reject any token not signed with HMAC. Without this check an attacker
		// could present an "alg: none" token, or one signed with a public key
		// they control, and have it accepted.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claims, nil
}

// subjectFrom builds the authorization Subject carried through the request.
func subjectFrom(claims *Claims) authz.Subject {
	return authz.Subject{
		UserID:      claims.UserID,
		IsSuperUser: claims.IsSuperUser,
	}
}

// Authenticate is net/http middleware that requires a valid token.
//
// Use it to protect a route group mounted outside the Huma API. Requests without
// credentials are rejected — for optional authentication, see HumaAuthenticate.
func Authenticate(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := parseBearerToken(r.Header.Get("Authorization"), jwtSecret)
			if err != nil {
				writeUnauthorized(w, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(authz.WithSubject(r.Context(), subjectFrom(claims))))
		})
	}
}

// HumaAuthenticate resolves identity for every request entering the Huma API.
//
// It distinguishes two cases that the previous implementation conflated:
//
//   - No Authorization header: the request proceeds anonymously. Public routes
//     work; protected routes are refused by their permission guard with 401.
//   - A header that is present but invalid, expired, or malformed: the request
//     is refused here with 401.
//
// That distinction matters to clients. Falling through with a bad token produced
// a 403 from the guard, which tells a client "you lack permission" when the truth
// is "your session expired" — so it never knew to refresh the token.
func HumaAuthenticate(api huma.API, jwtSecret string) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		claims, err := parseBearerToken(ctx.Header("Authorization"), jwtSecret)
		if err != nil {
			if errors.Is(err, ErrNoCredentials) {
				next(ctx) // anonymous — the guard decides whether that is allowed
				return
			}
			huma.WriteErr(api, ctx, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		reqCtx := authz.WithSubject(ctx.Context(), subjectFrom(claims))
		next(huma.WithContext(ctx, reqCtx))
	}
}

// RequireRole restricts access by the coarse Role claim.
//
// Prefer pkg/authz permissions for anything non-trivial: roles in a token cannot
// be revoked before expiry, and encoding authorization in the credential means
// changing what a user may do requires them to log in again.
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(roleKey{}).(string)
			if !allowed[role] {
				response.WriteJSON(w, http.StatusForbidden,
					`{"success":false,"message":"You do not have permission to perform this action"}`)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// roleKey carries the optional Role claim. Unexported so it cannot collide.
type roleKey struct{}

// WithRole stores the coarse role claim for RequireRole. Applications using
// pkg/authz permissions do not need this.
func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleKey{}, role)
}

// GetUserID returns the authenticated user's ID, or the nil UUID when the request
// is anonymous. Callers that need to distinguish the two should use
// authz.SubjectFromContext, which reports presence explicitly.
func GetUserID(ctx context.Context) uuid.UUID {
	sub, ok := authz.SubjectFromContext(ctx)
	if !ok {
		return uuid.Nil
	}
	return sub.UserID
}

func writeUnauthorized(w http.ResponseWriter, err error) {
	msg := "Invalid or expired token"
	if errors.Is(err, ErrNoCredentials) {
		msg = "Authorization header is required"
	}
	response.WriteJSON(w, http.StatusUnauthorized,
		fmt.Sprintf(`{"success":false,"message":%q}`, msg))
}

// TenantResolver builds a tenant.Resolver that reads the tenant from the token.
//
// A request with no credentials, or with a token carrying no tenant claim,
// resolves to no tenant rather than an error: public routes have to keep
// working, and a request that then touches a tenanted model fails closed at the
// query layer anyway. The failure surfaces where the data is, not at the edge
// where it would also break login.
//
// Wire it in only when the application has tenanted models:
//
//	router.Use(tenant.Middleware(middleware.TenantResolver(cfg.JWTSecret)))
func TenantResolver(jwtSecret string) tenant.Resolver {
	return func(r *http.Request) (uuid.UUID, error) {
		claims, err := parseBearerToken(r.Header.Get("Authorization"), jwtSecret)
		if err != nil {
			return uuid.Nil, nil
		}
		return claims.TenantID, nil
	}
}

// HumaTenantResolver is TenantResolver for the Huma API surface.
func HumaTenantResolver(jwtSecret string) tenant.HumaResolver {
	return func(ctx huma.Context) (uuid.UUID, error) {
		claims, err := parseBearerToken(ctx.Header("Authorization"), jwtSecret)
		if err != nil {
			return uuid.Nil, nil
		}
		return claims.TenantID, nil
	}
}
