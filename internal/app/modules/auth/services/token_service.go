package services

import (
	"errors"
	"fmt"
	"time"
appErrors "github.com/codetheuri/todolist/pkg/errors"
	"github.com/codetheuri/todolist/internal/app/modules/auth/models"
	"github.com/codetheuri/todolist/internal/app/modules/auth/repositories"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/net/context"
)

type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type TokenService interface {
	GenerateAuthTokens(ctx context.Context, user *models.User) (string, error)
	ValidateToken(ctx context.Context, tokenString string) (*Claims, error)
	RevokeToken(ctx context.Context, jti string, expiresAt time.Time) error
	IsTokenBlacklisted(ctx context.Context, jti string) (bool, error)
	CleanExpiredRevokedTokens(ctx context.Context) error
	GetTokenTTL() time.Time 
}

type tokenService struct {
	revokedTokenRepo repositories.RevokedTokenRepository
	log              logger.Logger
	jwtSecret        []byte
	tokenTTL         time.Duration
}

// constructor for the TokenService.
func NewTokenService(revokedTokenRepo repositories.RevokedTokenRepository, jwtSecret string, tokenTTL time.Duration, log logger.Logger) TokenService {
	return &tokenService{
		revokedTokenRepo: revokedTokenRepo,
		jwtSecret:        []byte(jwtSecret),
		tokenTTL:         tokenTTL,
		log:              log,
	}
}
func (s *tokenService) GenerateAuthTokens(ctx context.Context, user *models.User) (string, error) {
	s.log.Info("Generating JWT for user", "userID", user.ID, "email", user.Email)

	now := time.Now()
	expiresAt := now.Add(s.tokenTTL)
	jti := uuid.New().String() 

	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "tusk-api", 
			Subject:   fmt.Sprintf("%d", user.ID),
			ID:        jti, 
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		s.log.Error("Failed to sign JWT tokn", err, "userID", user.ID)
		return "", appErrors.InternalServerError("Failed to generate token", err)
	}

	s.log.Info("JWT generated successfully", "userID", user.ID, "jti", jti)
	return tokenString, nil
}
func (s *tokenService) GetTokenTTL() time.Time {
    return time.Now().Add(s.tokenTTL)
}
func (s *tokenService) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	s.log.Debug("Validating JWT token")

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method is what you expect
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, appErrors.AuthError(fmt.Sprintf("unexpected signing method: %v", token.Header["alg"]), nil)
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		s.log.Warn("Failed to parse or validate token", err)
		if errors.Is(err,jwt.ErrTokenExpired) {
			return nil, appErrors.AuthError("token expired", err)
		}
		if errors.Is(err, jwt.ErrSignatureInvalid) {
			return nil, appErrors.AuthError("invalid token signature", err)
		}
   
	
	return  nil, appErrors.InternalServerError("failed to parse token", err)
}

	if !token.Valid {
		s.log.Warn("Invalid token received")
		return nil, appErrors.AuthError("invalid token", nil)
	}

	// Crucially, check if the token's JTI is blacklisted
	isRevoked, err := s.revokedTokenRepo.IsTokenRevoked(ctx, claims.ID)
	if err != nil {
		s.log.Error("Failed to check if token is blacklisted", err, "jti", claims.ID)
		return nil, appErrors.DatabaseError("failed to check token revocation status", err)
	}
	if isRevoked {
		s.log.Warn("Token is blacklisted", "jti", claims.ID)
		return nil, appErrors.AuthError("token is blacklisted", nil)
	}

	s.log.Debug("Token validated successfully", "userID", claims.UserID, "jti", claims.ID)
	return claims, nil
}

func (s *tokenService) RevokeToken(ctx context.Context, jti string, expiresAt time.Time) error {
	s.log.Info("Revoking token", "jti", jti)

	revokedToken := &models.RevokedToken{
		JTI:       jti,
		ExpiresAt: expiresAt,
	}

	if err := s.revokedTokenRepo.SaveRevokedToken(ctx, revokedToken); err != nil {
		s.log.Error("Failed to save revoked token to DB", err, "jti", jti)
		return appErrors.DatabaseError("failed to revoke token", err)
	}
	s.log.Info("Token successfully revoked", "jti", jti)
	return nil
}

func (s *tokenService) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	s.log.Debug("Checking if token JTI is blacklisted directly", "jti", jti)
	   isRevoked, err := s.revokedTokenRepo.IsTokenRevoked(ctx, jti)
    if err != nil {
        s.log.Error("Failed to check if token is blacklisted via repo", err, "jti", jti)
        return false, appErrors.DatabaseError("failed to check token blacklist status", err) // <--- Use appErrors
    }
    return isRevoked, nil
}

func (s *tokenService) CleanExpiredRevokedTokens(ctx context.Context) error {
	s.log.Info("Initiating cleanup of expired revoked tokens")
	if err := s.revokedTokenRepo.DeleteExpiredRevokedTokens(ctx, time.Now()); err != nil {
		s.log.Error("Failed to clean up expired revoked tokens", err)
		return appErrors.DatabaseError("failed to clean up expired revoked tokens", err) // <--- Use appErrors
	}
	s.log.Info("Expired revoked tokens cleanup completed")
	return nil
}
