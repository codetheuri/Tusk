package authz

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/codetheuri/tusk/v2/pkg/id"
)

// PermissionRecord represents the runtime database representation of a code permission.
type PermissionRecord struct {
	Name        string    `gorm:"primaryKey;type:varchar(191)" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName explicitly overrides GORM table name convention for consistency.
func (PermissionRecord) TableName() string {
	return "permissions"
}

// Role represents an administrative role grouping multiple permissions.
type Role struct {
	ID          uuid.UUID          `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string             `gorm:"type:varchar(191);uniqueIndex;not null" json:"name"`
	Description string             `gorm:"type:text" json:"description"`
	Permissions []PermissionRecord `gorm:"many2many:role_permissions;foreignKey:ID;joinForeignKey:role_id;references:Name;joinReferences:permission_name" json:"permissions,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// RolePermission links a Role to a PermissionRecord.
type RolePermission struct {
	RoleID         uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"role_id"`
	PermissionName string    `gorm:"primaryKey;type:varchar(191);index" json:"permission_name"`
	CreatedAt      time.Time `json:"created_at"`
}

// BeforeCreate assigns an identifier when one was not supplied.
func (r *Role) BeforeCreate(*gorm.DB) error {
	if id.IsZero(r.ID) {
		r.ID = id.New()
	}
	return nil
}

// TableName explicitly sets table name for RolePermission.
func (RolePermission) TableName() string {
	return "role_permissions"
}

// UserRole links a User ID to a Role ID.
type UserRole struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"user_id"`
	RoleID    uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName explicitly sets table name for UserRole.
func (UserRole) TableName() string {
	return "user_roles"
}
