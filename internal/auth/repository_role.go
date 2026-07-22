package auth

import (
	"context"

	"github.com/codetheuri/tusk/pkg/authz"
	"gorm.io/gorm"
)

// CreateRole inserts a new role into the database.
func (r *Repository) CreateRole(ctx context.Context, role *Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

// ListRoles fetches all roles along with their permission names.
func (r *Repository) ListRoles(ctx context.Context) ([]Role, error) {
	var roles []Role
	if err := r.db.WithContext(ctx).Find(&roles).Error; err != nil {
		return nil, err
	}

	for i := range roles {
		var perms []string
		r.db.WithContext(ctx).Model(&RolePermission{}).Where("role_id = ?", roles[i].ID).Pluck("permission_name", &perms)
		roles[i].Permissions = perms
	}

	return roles, nil
}

// GetRoleByID fetches a role by its ID along with its permissions.
func (r *Repository) GetRoleByID(ctx context.Context, id uint) (*Role, error) {
	var role Role
	if err := r.db.WithContext(ctx).First(&role, id).Error; err != nil {
		return nil, err
	}

	var perms []string
	r.db.WithContext(ctx).Model(&RolePermission{}).Where("role_id = ?", role.ID).Pluck("permission_name", &perms)
	role.Permissions = perms

	return &role, nil
}

// UpdateRole updates role name and description.
func (r *Repository) UpdateRole(ctx context.Context, role *Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

// DeleteRole deletes a role and cascades removal from join tables.
func (r *Repository) DeleteRole(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", id).Delete(&RolePermission{}).Error; err != nil {
			return err
		}
		if err := tx.Where("role_id = ?", id).Delete(&UserRole{}).Error; err != nil {
			return err
		}
		return tx.Delete(&Role{}, id).Error
	})
}

// EnsurePermission guarantees a registered permission exists in the database permissions table.
func (r *Repository) EnsurePermission(ctx context.Context, name, description string) error {
	p := authz.PermissionRecord{
		Name:        name,
		Description: description,
	}
	return r.db.WithContext(ctx).FirstOrCreate(&p, authz.PermissionRecord{Name: name}).Error
}

// AddRolePermission links a permission string to a role.
func (r *Repository) AddRolePermission(ctx context.Context, roleID uint, permName string) error {
	rp := RolePermission{
		RoleID:         roleID,
		PermissionName: permName,
	}
	return r.db.WithContext(ctx).FirstOrCreate(&rp, RolePermission{RoleID: roleID, PermissionName: permName}).Error
}

// RemoveRolePermission removes a permission string from a role.
func (r *Repository) RemoveRolePermission(ctx context.Context, roleID uint, permName string) error {
	return r.db.WithContext(ctx).Where("role_id = ? AND permission_name = ?", roleID, permName).Delete(&RolePermission{}).Error
}

// AssignUserRole links a user to a specific role.
func (r *Repository) AssignUserRole(ctx context.Context, userID uint, roleID uint) error {
	ur := UserRole{
		UserID: userID,
		RoleID: roleID,
	}
	return r.db.WithContext(ctx).FirstOrCreate(&ur, UserRole{UserID: userID, RoleID: roleID}).Error
}

// RemoveUserRole revokes a role from a user.
func (r *Repository) RemoveUserRole(ctx context.Context, userID uint, roleID uint) error {
	return r.db.WithContext(ctx).Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&UserRole{}).Error
}

// GetUserPermissions returns all permission strings granted to a user via assigned roles.
func (r *Repository) GetUserPermissions(ctx context.Context, userID uint) ([]string, error) {
	var perms []string
	err := r.db.WithContext(ctx).
		Table("role_permissions").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").
		Where("user_roles.user_id = ?", userID).
		Pluck("role_permissions.permission_name", &perms).Error
	return perms, err
}
