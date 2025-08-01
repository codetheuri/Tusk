package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	tokenPkg "github.com/codetheuri/todolist/pkg/auth/token"
	appErrors "github.com/codetheuri/todolist/pkg/errors"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/codetheuri/todolist/pkg/web"
)

// --- authenticator middleware to validate jwt tokens

func Authenticator(tokenService tokenPkg.TokenService, log logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Debug("Middleware: Authenticator invoked")
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				web.RespondError(w, appErrors.AuthError("Authorization header is required", nil), http.StatusUnauthorized)
				return
			}

			// expect "Bearer TOKEN"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				web.RespondError(w, appErrors.AuthError("Authorization header format must be 'Bearer <token>'", nil), http.StatusUnauthorized)
				return
			}
			tokenString := parts[1]
			ctx := r.Context()
			claims, err := tokenService.ValidateToken(ctx, tokenString)
			if err != nil {
				log.Warn("Middleware: Token validation failed", err)
				var appErr appErrors.AppError
				if errors.As(err, &appErr) {
					// web.RespondError(w, appErr, http.StatusUnauthorized)
					web.RespondMessage(w, http.StatusUnauthorized, appErr.Message(), "", "")
				} else {
					web.RespondError(w, appErrors.AuthError("Your request was made with invalid credentials.", err), http.StatusUnauthorized)
				}
				return
			}
			// Token is valid. Inject UserID and Role into the request context.

			ctx = context.WithValue(ctx, tokenPkg.ContextKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, tokenPkg.ContextKeyUserRole, claims.Role)
			ctx = context.WithValue(ctx, tokenPkg.ContextKeyJTI, claims.ID)
			ctx = context.WithValue(ctx, tokenPkg.ContextKeyExpiresAt, claims.ExpiresAt.Time)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)

		})
	}

}



//retrieve role from urequest context

func GetRoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(tokenPkg.ContextKeyUserRole).(string)
	return role, ok
}

func Authorizer(requiredRoles ...string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole, ok := GetRoleFromContext(r.Context())
			if !ok {
				web.RespondError(w, appErrors.InternalServerError("User role not found in context", nil), http.StatusInternalServerError)
				return
			}

			hasPermission := false
			for _, requiredRole := range requiredRoles {
				if userRole == requiredRole {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				web.RespondError(w, appErrors.AuthorizationError("You do not have permission to access this resource", nil), http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
