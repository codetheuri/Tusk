package authz

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// Subject represents the authenticated user context performing an action.
type Subject struct {
	UserID      uint
	IsSuperUser bool
}

// Policy defines an authorization evaluation rule.
// This interface allows future expansion to ABAC (Attribute-Based Access Control)
// or dynamic policy evaluation without modifying endpoint handlers.
type Policy interface {
	Evaluate(ctx context.Context, db *gorm.DB, sub Subject) (bool, error)
}

// Evaluator checks permissions against subjects using the database.
type Evaluator struct {
	db *gorm.DB
}

// NewEvaluator creates a new authorization evaluator.
func NewEvaluator(db *gorm.DB) *Evaluator {
	return &Evaluator{db: db}
}

// IsAuthorized evaluates whether a subject satisfies the given policy.
// Super Users bypass all checks immediately.
func (e *Evaluator) IsAuthorized(ctx context.Context, sub Subject, policy Policy) (bool, error) {
	if sub.IsSuperUser {
		return true, nil
	}
	return policy.Evaluate(ctx, e.db, sub)
}

// UserHasPermission queries whether a user possesses a specific permission through any assigned role.
func (e *Evaluator) UserHasPermission(ctx context.Context, userID uint, permission string) (bool, error) {
	var count int64
	err := e.db.WithContext(ctx).Table("user_roles").
		Joins("JOIN role_permissions ON role_permissions.role_id = user_roles.role_id").
		Where("user_roles.user_id = ? AND role_permissions.permission_name = ?", userID, permission).
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("failed to evaluate user permission: %w", err)
	}

	return count > 0, nil
}

// --- Built-in Policy Implementations ---

// RequirePermissionPolicy demands a single specific permission.
type RequirePermissionPolicy struct {
	Permission string
}

func (p RequirePermissionPolicy) Evaluate(ctx context.Context, db *gorm.DB, sub Subject) (bool, error) {
	eval := NewEvaluator(db)
	return eval.UserHasPermission(ctx, sub.UserID, p.Permission)
}

// RequireAnyPolicy demands at least one of the specified permissions.
type RequireAnyPolicy struct {
	Permissions []string
}

func (p RequireAnyPolicy) Evaluate(ctx context.Context, db *gorm.DB, sub Subject) (bool, error) {
	if len(p.Permissions) == 0 {
		return true, nil
	}

	var count int64
	err := db.WithContext(ctx).Table("user_roles").
		Joins("JOIN role_permissions ON role_permissions.role_id = user_roles.role_id").
		Where("user_roles.user_id = ? AND role_permissions.permission_name IN ?", sub.UserID, p.Permissions).
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("failed to evaluate RequireAny policy: %w", err)
	}

	return count > 0, nil
}

// RequireAllPolicy demands that the subject possesses ALL specified permissions.
type RequireAllPolicy struct {
	Permissions []string
}

func (p RequireAllPolicy) Evaluate(ctx context.Context, db *gorm.DB, sub Subject) (bool, error) {
	if len(p.Permissions) == 0 {
		return true, nil
	}

	var matchedCount int64
	err := db.WithContext(ctx).Table("user_roles").
		Joins("JOIN role_permissions ON role_permissions.role_id = user_roles.role_id").
		Where("user_roles.user_id = ? AND role_permissions.permission_name IN ?", sub.UserID, p.Permissions).
		Select("COUNT(DISTINCT role_permissions.permission_name)").
		Count(&matchedCount).Error

	if err != nil {
		return false, fmt.Errorf("failed to evaluate RequireAll policy: %w", err)
	}

	return int(matchedCount) == len(p.Permissions), nil
}
