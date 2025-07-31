package token

import (
	"context"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type TokenService interface {
	GenerateToken(userID string, role string) (string, error)
	ValidateToken(ctx context.Context, tokenString string) (*Claims, error)
	RevokeToken(ctx context.Context, jti string, expiresAt time.Time) error
	IsTokenBlacklisted(ctx context.Context, jti string) (bool, error)
	CleanExpiredRevokedTokens(ctx context.Context) error

	GetTokenTTL() time.Time
}

// type RevokedToken struct{
// 	JTI       string    `json:"jti" gorm:"primaryKey"`
// 	ExpiresAt time.Time `gorm:"index"`
// }
// // RevokedTokenRepository interface also if needed by TokenService's interface contract
// type RevokedTokenRepository interface {
//     SaveRevokedToken(ctx context.Context, token *RevokedToken) error
//     IsTokenRevoked(ctx context.Context, jti string) (bool, error)
//     DeleteExpiredRevokedTokens(ctx context.Context, now time.Time) error
// }

type contextKey string

const (
	ContextKeyUserID    contextKey = "userID"
	ContextKeyUserRole  contextKey = "userRole"
	ContextKeyJTI       contextKey = "jti"
	ContextKeyExpiresAt contextKey = "expiresAt"
)

// retrieve the userID from the request context
func GetUserIDFromContext(ctx context.Context) (uint, bool) {
	val := ctx.Value(ContextKeyUserID)
	if idStr, ok := val.(string); ok {
		if id, err := strconv.ParseUint(idStr, 10, 64); err == nil {
			if id <= uint64(^uint(0)) {
				return uint(id), true
			}
		}
	}
	return 0, false
}

// retrieve user role from the request context
func GetuserRoleFromContext(ctx context.Context) (string, bool) {
	val := ctx.Value(ContextKeyUserRole)
	role, ok := val.(string)
	return role, ok
}

// retrieve the JTI from the request context
func GetJTIFromContext(ctx context.Context) (string, bool) {
	val := ctx.Value(ContextKeyJTI)
	jti, ok := val.(string)
	return jti, ok
}

// retrieve the expiresAt from the request context
func GetExpiresAtFromContext(ctx context.Context) (time.Time, bool) {
	val := ctx.Value(ContextKeyExpiresAt)
	exp, ok := val.(time.Time)
	return exp, ok
}
