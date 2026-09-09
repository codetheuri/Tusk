package auth

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/codetheuri/tusk/v2/pkg/id"
)

// User handles core authentication data, credentials, and security state.
type User struct {
	ID                  uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	Username            string     `json:"username" gorm:"uniqueIndex;not null"`
	Email               string     `json:"email" gorm:"uniqueIndex;not null"`
	Phone               *string    `json:"phone,omitempty" gorm:"uniqueIndex"`
	Password            string     `json:"-" gorm:"not null"`
	IsSuperUser         bool       `json:"is_super_user" gorm:"default:false"`
	IsActive            bool       `json:"is_active" gorm:"default:true"`
	IsVerified          bool       `json:"is_verified" gorm:"default:false"`
	FailedLoginAttempts int        `json:"failed_login_attempts" gorm:"default:0"`
	LockedUntil         *time.Time `json:"locked_until,omitempty"`
	LastLoginAt         *time.Time `json:"last_login_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`

	// 1-to-1 Profile relationship
	Profile *UserProfile `json:"profile,omitempty" gorm:"foreignKey:UserID"`
}

// BeforeCreate assigns an identifier when the caller has not supplied one.
//
// The zero check is the important part, not the assignment: a client that
// generated its own ID while offline must keep it, because that ID may already be
// referenced by other records it created before it could reach the server.
// Overwriting it here would break those references silently.
func (u *User) BeforeCreate(*gorm.DB) error {
	if id.IsZero(u.ID) {
		u.ID = id.New()
	}
	return nil
}

// UserProfile stores personal identity information.
type UserProfile struct {
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;primaryKey"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Avatar    string    `json:"avatar"`
	Bio       string    `json:"bio"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RefreshToken stores hashed refresh tokens for session management and revocation.
type RefreshToken struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	TokenHash string     `json:"-" gorm:"uniqueIndex;not null"`
	ExpiresAt time.Time  `json:"expires_at" gorm:"not null"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

func (r *RefreshToken) BeforeCreate(*gorm.DB) error {
	if id.IsZero(r.ID) {
		r.ID = id.New()
	}
	return nil
}

// Role represents a security role containing permissions.
type Role struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	Name        string    `json:"name" gorm:"uniqueIndex;not null"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Permissions []string  `json:"permissions,omitempty" gorm:"-"`
}

func (r *Role) BeforeCreate(*gorm.DB) error {
	if id.IsZero(r.ID) {
		r.ID = id.New()
	}
	return nil
}

// RolePermission defines the join table linking roles to permissions.
type RolePermission struct {
	RoleID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	PermissionName string    `gorm:"primaryKey"`
	CreatedAt      time.Time
}

// UserRole defines the join table linking users to roles.
type UserRole struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	RoleID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	CreatedAt time.Time
}
