package auth

import (
	"context"
	"fmt"

	"github.com/codetheuri/tusk/pkg/authz"
)

type CreateRoleRequest struct {
	Name        string
	Description string
}

type UpdateRoleRequest struct {
	ID          uint
	Name        *string
	Description *string
}

// CreateRole creates a new security role without permissions attached initially.
func (s *Service) CreateRole(ctx context.Context, req *CreateRoleRequest) (*Role, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("role name cannot be empty")
	}

	role := &Role{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := s.repo.CreateRole(ctx, role); err != nil {
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	return role, nil
}

// ListRoles returns all roles in the system.
func (s *Service) ListRoles(ctx context.Context) ([]Role, error) {
	return s.repo.ListRoles(ctx)
}

// GetRoleByID returns a single role by ID.
func (s *Service) GetRoleByID(ctx context.Context, id uint) (*Role, error) {
	return s.repo.GetRoleByID(ctx, id)
}

// UpdateRole updates a role's name or description safely using partial updates.
func (s *Service) UpdateRole(ctx context.Context, req *UpdateRoleRequest) (*Role, error) {
	role, err := s.repo.GetRoleByID(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("role not found")
	}

	if req.Name != nil && *req.Name != "" {
		role.Name = *req.Name
	}
	if req.Description != nil {
		role.Description = *req.Description
	}

	if err := s.repo.UpdateRole(ctx, role); err != nil {
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	return role, nil
}

// DeleteRole removes a role from the system.
func (s *Service) DeleteRole(ctx context.Context, id uint) error {
	return s.repo.DeleteRole(ctx, id)
}

// AddRolePermission attaches a permission string to a role.
func (s *Service) AddRolePermission(ctx context.Context, roleID uint, permName string) error {
	perm, exists := authz.DefaultRegistry().Find(permName)
	if !exists {
		return fmt.Errorf("permission '%s' is not a valid system permission", permName)
	}

	if err := s.repo.EnsurePermission(ctx, perm.Name, perm.Description); err != nil {
		return fmt.Errorf("failed to sync permission to database: %w", err)
	}

	return s.repo.AddRolePermission(ctx, roleID, permName)
}

// RemoveRolePermission detaches a permission string from a role.
func (s *Service) RemoveRolePermission(ctx context.Context, roleID uint, permName string) error {
	return s.repo.RemoveRolePermission(ctx, roleID, permName)
}

// AssignUserRole assigns a role to a user after checking existence.
func (s *Service) AssignUserRole(ctx context.Context, userID uint, roleID uint) error {
	if _, err := s.repo.GetRoleByID(ctx, roleID); err != nil {
		return fmt.Errorf("role with ID %d does not exist", roleID)
	}
	if _, err := s.repo.FindByID(ctx, userID); err != nil {
		return fmt.Errorf("user with ID %d does not exist", userID)
	}
	return s.repo.AssignUserRole(ctx, userID, roleID)
}

// RemoveUserRole revokes a role from a user.
func (s *Service) RemoveUserRole(ctx context.Context, userID uint, roleID uint) error {
	return s.repo.RemoveUserRole(ctx, userID, roleID)
}
