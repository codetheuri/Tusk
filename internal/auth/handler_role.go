package auth

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/codetheuri/tusk/pkg/response"
)

// CreateRole handles creating a new RBAC role (name & description).
func (h *Handler) CreateRole(ctx context.Context, input *CreateRoleInput) (*RoleOutput, error) {
	req := &CreateRoleRequest{
		Name:        input.Body.Name,
		Description: input.Body.Description,
	}

	role, err := h.service.CreateRole(ctx, req)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to create role", err)
	}

	resp := &RoleOutput{}
	resp.Body.Success = true
	resp.Body.Message = "Role created successfully"
	resp.Body.Data.Role = role
	return resp, nil
}

// ListRoles handles fetching all registered roles.
func (h *Handler) ListRoles(ctx context.Context, input *ListRolesInput) (*RolesOutput, error) {
	roles, err := h.service.ListRoles(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to retrieve roles", err)
	}

	resp := &RolesOutput{}
	resp.Body.Success = true
	resp.Body.Message = "Roles retrieved successfully"
	resp.Body.Data.Roles = roles
	return resp, nil
}

// GetRole handles retrieving a single role by ID.
func (h *Handler) GetRole(ctx context.Context, input *RoleIDInput) (*RoleOutput, error) {
	role, err := h.service.GetRoleByID(ctx, input.ID)
	if err != nil {
		return nil, huma.Error404NotFound("Role not found", err)
	}

	resp := &RoleOutput{}
	resp.Body.Success = true
	resp.Body.Message = "Role retrieved successfully"
	resp.Body.Data.Role = role
	return resp, nil
}

// UpdateRole handles updating a role's name and description.
func (h *Handler) UpdateRole(ctx context.Context, input *UpdateRoleInput) (*RoleOutput, error) {
	req := &UpdateRoleRequest{
		ID:          input.ID,
		Name:        input.Body.Name,
		Description: input.Body.Description,
	}

	role, err := h.service.UpdateRole(ctx, req)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to update role", err)
	}

	resp := &RoleOutput{}
	resp.Body.Success = true
	resp.Body.Message = "Role updated successfully"
	resp.Body.Data.Role = role
	return resp, nil
}

// DeleteRole handles deleting a role.
func (h *Handler) DeleteRole(ctx context.Context, input *RoleIDInput) (*MessageOutput, error) {
	if err := h.service.DeleteRole(ctx, input.ID); err != nil {
		return nil, huma.Error400BadRequest("Failed to delete role", err)
	}

	resp := &MessageOutput{}
	resp.Body.Success = true
	resp.Body.Message = "Role deleted successfully"
	return resp, nil
}

// AddRolePermission handles attaching a permission string to a role.
func (h *Handler) AddRolePermission(ctx context.Context, input *AddRolePermissionInput) (*MessageOutput, error) {
	if err := h.service.AddRolePermission(ctx, input.ID, input.Body.PermissionName); err != nil {
		return nil, response.FieldError("Validation failed", "permission_name", err.Error())
	}

	resp := &MessageOutput{}
	resp.Body.Success = true
	resp.Body.Message = "Permission added to role successfully"
	return resp, nil
}

// RemoveRolePermission handles detaching a permission string from a role.
func (h *Handler) RemoveRolePermission(ctx context.Context, input *RemoveRolePermissionInput) (*MessageOutput, error) {
	if err := h.service.RemoveRolePermission(ctx, input.ID, input.PermissionName); err != nil {
		return nil, huma.Error400BadRequest("Failed to remove permission from role", err)
	}

	resp := &MessageOutput{}
	resp.Body.Success = true
	resp.Body.Message = "Permission removed from role successfully"
	return resp, nil
}

// AssignUserRole handles assigning a role to a user.
func (h *Handler) AssignUserRole(ctx context.Context, input *AssignUserRoleInput) (*MessageOutput, error) {
	if err := h.service.AssignUserRole(ctx, input.UserID, input.Body.RoleID); err != nil {
		return nil, response.FieldError("Validation failed", "role_id", err.Error())
	}

	resp := &MessageOutput{}
	resp.Body.Success = true
	resp.Body.Message = "Role assigned to user successfully"
	return resp, nil
}

// RemoveUserRole handles revoking a role from a user.
func (h *Handler) RemoveUserRole(ctx context.Context, input *RemoveUserRoleInput) (*MessageOutput, error) {
	if err := h.service.RemoveUserRole(ctx, input.UserID, input.RoleID); err != nil {
		return nil, huma.Error400BadRequest("Failed to revoke role from user", err)
	}

	resp := &MessageOutput{}
	resp.Body.Success = true
	resp.Body.Message = "Role revoked from user successfully"
	return resp, nil
}
