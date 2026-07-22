package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type contextKey string

const (
	ContextKeyUserID contextKey = "user_id"
	ContextKeyRole   contextKey = "role"
	ContextKeyJTI    contextKey = "jti"
)

type Claims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func Authenticate(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// To keep it clean, we'd normally call a helper. 
				// We can just use http.Error or a custom response.
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"success":false,"error":{"code":"UNAUTHORIZED","message":"Authorization header is required"}}`))
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"success":false,"error":{"code":"UNAUTHORIZED","message":"Invalid authorization format. Expected: Bearer <token>"}}`))
				return
			}

			tokenString := parts[1]
			claims := &Claims{}

			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"success":false,"error":{"code":"UNAUTHORIZED","message":"Invalid or expired token"}}`))
				return
			}

			ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.UserID)
			ctx = context.WithValue(ctx, "user_id", claims.UserID)
			ctx = context.WithValue(ctx, ContextKeyRole, claims.Role)
			ctx = context.WithValue(ctx, "role", claims.Role)
			ctx = context.WithValue(ctx, ContextKeyJTI, claims.ID)
			ctx = context.WithValue(ctx, "jti", claims.ID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	allowedSet := make(map[string]bool, len(allowedRoles))
	for _, r := range allowedRoles {
		allowedSet[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value(ContextKeyRole).(string)
			if !ok || !allowedSet[role] {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"success":false,"error":{"code":"FORBIDDEN","message":"You do not have permission to perform this action"}}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func GetUserID(ctx context.Context) uint {
	id, _ := ctx.Value(ContextKeyUserID).(uint)
	return id
}

// HumaAuthenticate decodes the JWT Authorization header into the Huma request context.
func HumaAuthenticate(api huma.API, jwtSecret string, db *gorm.DB) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		authHeader := ctx.Header("Authorization")
		if authHeader == "" {
			next(ctx)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			tokenString := parts[1]
			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})

			if err == nil && token.Valid {
				reqCtx := ctx.Context()
				reqCtx = context.WithValue(reqCtx, ContextKeyUserID, claims.UserID)
				reqCtx = context.WithValue(reqCtx, "user_id", claims.UserID)
				reqCtx = context.WithValue(reqCtx, ContextKeyRole, claims.Role)
				reqCtx = context.WithValue(reqCtx, "role", claims.Role)

				var isSuper bool
				db.WithContext(reqCtx).Table("users").Where("id = ?", claims.UserID).Pluck("is_super_user", &isSuper)
				reqCtx = context.WithValue(reqCtx, "is_super_user", isSuper)

				ctx = huma.WithContext(ctx, reqCtx)
			}
		}

		next(ctx)
	}
}
