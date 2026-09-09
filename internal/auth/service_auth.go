package auth

import (
	"github.com/google/uuid"

	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/codetheuri/tusk/config"
	"github.com/codetheuri/tusk/pkg/authz"
	"github.com/codetheuri/tusk/pkg/middleware"
	"github.com/codetheuri/tusk/pkg/query"
)

// Service encapsulates the business logic for Identity, Auth, and RBAC.
type Service struct {
	repo *Repository
	cfg  *config.Config
}

func NewService(repo *Repository, cfg *config.Config) *Service {
	return &Service{repo: repo, cfg: cfg}
}

// Register creates a new user account and associated identity profile.
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*User, error) {
	if req.Password != req.PasswordConfirm {
		return nil, fmt.Errorf("passwords do not match")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &User{
		Username:   req.Username,
		Email:      req.Email,
		Phone:      req.Phone,
		Password:   string(hash),
		IsActive:   true,
		IsVerified: false,
	}

	profile := &UserProfile{
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}

	if err := s.repo.CreateUserWithProfile(ctx, user, profile); err != nil {
		return nil, fmt.Errorf("username, email, or phone already exists")
	}

	return user, nil
}

type AuthTokens struct {
	User         *User
	AccessToken  string
	RefreshToken string
}

// Login authenticates a user via single flexible login field (username, email, or phone).
func (s *Service) Login(ctx context.Context, req *LoginRequest) (*AuthTokens, error) {
	user, err := s.repo.FindByLogin(ctx, req.Login)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// 1. Account Status Check
	if !user.IsActive {
		return nil, fmt.Errorf("account deactivated: please contact support")
	}

	// 2. Lockout Check
	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, fmt.Errorf("account locked due to multiple failed login attempts: try again later")
	}

	// 3. Password Verification
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		user.FailedLoginAttempts++
		if user.FailedLoginAttempts >= 5 {
			lockout := time.Now().Add(15 * time.Minute)
			user.LockedUntil = &lockout
		}
		_ = s.repo.UpdateUser(ctx, user)
		return nil, fmt.Errorf("invalid credentials")
	}

	// Reset failed attempts & set last login timestamp
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	now := time.Now()
	user.LastLoginAt = &now
	_ = s.repo.UpdateUser(ctx, user)

	// 4. Token Generation (Access JWT + Opaque Refresh Token)
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.issueRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to issue refresh token: %w", err)
	}

	return &AuthTokens{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// RefreshToken implements token rotation by validating a refresh token, revoking it, and issuing new tokens.
func (s *Service) RefreshToken(ctx context.Context, rawRefreshToken string) (*AuthTokens, error) {
	hash := s.hashToken(rawRefreshToken)

	tokenRecord, err := s.repo.FindRefreshToken(ctx, hash)
	if err != nil || tokenRecord.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("invalid or expired refresh token")
	}

	// Revoke old refresh token (Token Rotation!)
	_ = s.repo.RevokeRefreshToken(ctx, hash)

	user, err := s.repo.FindByID(ctx, tokenRecord.UserID)
	if err != nil || !user.IsActive {
		return nil, fmt.Errorf("user account invalid or deactivated")
	}

	newAccessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := s.issueRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to issue refresh token: %w", err)
	}

	return &AuthTokens{
		User:         user,
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

// Logout revokes a refresh token session.
func (s *Service) Logout(ctx context.Context, rawRefreshToken string) error {
	hash := s.hashToken(rawRefreshToken)
	return s.repo.RevokeRefreshToken(ctx, hash)
}

// GetCurrentUser returns the user model (with preloaded Profile) and permissions for the authenticated user.
func (s *Service) GetCurrentUser(ctx context.Context, userID uuid.UUID) (*User, []string, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("user not found: %w", err)
	}

	perms, err := s.repo.GetUserPermissions(ctx, userID)
	if err != nil {
		perms = []string{}
	}

	return user, perms, nil
}

// UpdateProfile updates the authenticated user's personal identity profile details.
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, firstName, lastName, avatar, bio string) (*UserProfile, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	profile := user.Profile
	if profile == nil {
		profile = &UserProfile{UserID: userID}
	}

	profile.FirstName = firstName
	profile.LastName = lastName
	profile.Avatar = avatar
	profile.Bio = bio

	if err := s.repo.UpdateProfile(ctx, profile); err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	return profile, nil
}

// ListUsers returns paginated user accounts matching query parameter criteria.
func (s *Service) ListUsers(ctx context.Context, q query.Query) ([]User, query.Meta, error) {
	return s.repo.ListUsers(ctx, q)
}

// ListPermissions returns all registered permissions in the system.
func (s *Service) ListPermissions(ctx context.Context) ([]authz.Permission, error) {
	return authz.DefaultRegistry().All(), nil
}

// --- Helper Functions ---

func (s *Service) generateAccessToken(user *User) (string, error) {
	expiry := time.Now().Add(s.cfg.AccessTokenTTL)
	claims := middleware.Claims{
		UserID: user.ID,
		// Carried in the token so authorization does not need a database round-trip
		// on every request. See middleware.Claims for the revocation tradeoff.
		IsSuperUser: user.IsSuperUser,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

func (s *Service) issueRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	rawToken := hex.EncodeToString(bytes)
	tokenHash := s.hashToken(rawToken)

	refreshToken := &RefreshToken{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 Days TTL
	}

	if err := s.repo.CreateRefreshToken(ctx, refreshToken); err != nil {
		return "", err
	}

	return rawToken, nil
}

func (s *Service) hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
