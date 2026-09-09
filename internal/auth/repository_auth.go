package auth

import (
	"github.com/google/uuid"

	"context"
	"time"

	"github.com/codetheuri/tusk/v2/pkg/logger"
	"github.com/codetheuri/tusk/v2/pkg/query"
	"gorm.io/gorm"
)

type Repository struct {
	db  *gorm.DB
	log logger.Logger
}

func NewRepository(db *gorm.DB, log logger.Logger) *Repository {
	return &Repository{
		db:  db,
		log: log,
	}
}

// CreateUserWithProfile inserts a new user and user profile inside a transaction.
func (r *Repository) CreateUserWithProfile(ctx context.Context, user *User, profile *UserProfile) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		profile.UserID = user.ID
		if err := tx.Create(profile).Error; err != nil {
			return err
		}
		user.Profile = profile
		return nil
	})
}

// FindByLogin searches for a user by username, email, or phone number.
func (r *Repository) FindByLogin(ctx context.Context, login string) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).
		Preload("Profile").
		Where("username = ? OR email = ? OR phone = ?", login, login, login).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID fetches a user by primary key ID with Profile preloaded.
func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).Preload("Profile").First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUser saves changes to a user model (security status, last login, lockout status).
func (r *Repository) UpdateUser(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// UpdateProfile updates a user's personal identity profile details.
func (r *Repository) UpdateProfile(ctx context.Context, profile *UserProfile) error {
	return r.db.WithContext(ctx).Save(profile).Error
}

// ListUsers fetches paginated users with preloaded profiles using pkg/query.
func (r *Repository) ListUsers(ctx context.Context, q query.Query) ([]User, query.Meta, error) {
	cfg := query.Config{
		DefaultSort:    "-created_at",
		DefaultPerPage: 20,
		MaxPerPage:     100,
		AllowedSorts: map[string]string{
			"id":         "users.id",
			"username":   "users.username",
			"email":      "users.email",
			"created_at": "users.created_at",
		},
		AllowedSearches: []string{"users.username", "users.email", "users.phone"},
		AllowedFilters: map[string]string{
			"is_active":   "users.is_active",
			"is_verified": "users.is_verified",
		},
	}

	return query.Paginate[User](ctx, r.db.Preload("Profile"), q, cfg)
}

// Refresh Token Storage & Management

func (r *Repository) CreateRefreshToken(ctx context.Context, token *RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *Repository) FindRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	var token RefreshToken
	if err := r.db.WithContext(ctx).Where("token_hash = ? AND revoked_at IS NULL", tokenHash).First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *Repository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&RefreshToken{}).
		Where("token_hash = ?", tokenHash).
		Update("revoked_at", &now).Error
}
