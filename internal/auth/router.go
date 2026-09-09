package auth

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"gorm.io/gorm"

	"github.com/codetheuri/tusk/v2/config"
	"github.com/codetheuri/tusk/v2/pkg/authz"
	"github.com/codetheuri/tusk/v2/pkg/logger"
)

func RegisterRoutes(api huma.API, db *gorm.DB, cfg *config.Config, log logger.Logger) {
	repo := NewRepository(db, log)
	service := NewService(repo, cfg)
	handler := NewHandler(service, log)

	// Auth Guard for route authorization
	guard := authz.NewGuard(api, db)

	// -------------------------------------------------------------
	// PUBLIC AUTHENTICATION ENDPOINTS
	// -------------------------------------------------------------

	huma.Register(api, huma.Operation{
		OperationID: "register-user",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/register",
		Summary:     "Register a new user",
		Description: "Creates a new user account and personal identity profile.",
		Tags:        []string{"Authentication"},
	}, handler.Register)

	huma.Register(api, huma.Operation{
		OperationID: "login-user",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/login",
		Summary:     "Login user",
		Description: "Authenticates a user via flexible single field (username, email, or phone) and returns Access + Refresh tokens.",
		Tags:        []string{"Authentication"},
	}, handler.Login)

	huma.Register(api, huma.Operation{
		OperationID: "refresh-token",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/refresh",
		Summary:     "Refresh Access Token",
		Description: "Rotates the Refresh Token and issues a new Access Token pair.",
		Tags:        []string{"Authentication"},
	}, handler.RefreshToken)

	huma.Register(api, huma.Operation{
		OperationID: "logout-user",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/logout",
		Summary:     "Logout user",
		Description: "Revokes the active Refresh Token session.",
		Tags:        []string{"Authentication"},
	}, handler.Logout)

	// -------------------------------------------------------------
	// PROTECTED AUTHENTICATION & PROFILE ENDPOINTS
	// -------------------------------------------------------------

	huma.Register(api, guard.Protected(huma.Operation{
		OperationID: "get-me",
		Method:      http.MethodGet,
		Path:        "/api/v1/auth/me",
		Summary:     "Current user profile",
		Description: "Returns details, identity profile, and granted permissions for the authenticated user.",
		Tags:        []string{"Authentication"},
	}, ""), handler.Me)

	huma.Register(api, guard.Protected(huma.Operation{
		OperationID: "update-my-profile",
		Method:      http.MethodPut,
		Path:        "/api/v1/auth/me/profile",
		Summary:     "Update profile",
		Description: "Updates personal identity information (first_name, last_name, avatar, bio).",
		Tags:        []string{"Authentication"},
	}, ""), handler.UpdateProfile)

	huma.Register(api, guard.Protected(huma.Operation{
		OperationID: "list-users",
		Method:      http.MethodGet,
		Path:        "/api/v1/auth/users",
		Summary:     "List users",
		Description: "Returns all user accounts in the system. Requires users.read permission.",
		Tags:        []string{"Authentication"},
	}, PermUsersRead), handler.ListUsers)

	huma.Register(api, guard.Protected(huma.Operation{
		OperationID: "list-permissions",
		Method:      http.MethodGet,
		Path:        "/api/v1/auth/permissions",
		Summary:     "List permissions",
		Description: "Returns all developer-defined system permissions registered across modules. Requires permissions.read.",
		Tags:        []string{"Authorization"},
	}, PermPermissionsRead), handler.ListPermissions)

	// -------------------------------------------------------------
	// ROLE & PERMISSION MANAGEMENT ENDPOINTS
	// -------------------------------------------------------------

	huma.Register(api, guard.Protected(huma.Operation{
		OperationID: "create-role",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/roles",
		Summary:     "Create role",
		Description: "Creates a new security role with name and description.",
		Tags:        []string{"Authorization"},
	}, PermRolesCreate), handler.CreateRole)

	huma.Register(api, guard.Protected(huma.Operation{
		OperationID: "list-roles",
		Method:      http.MethodGet,
		Path:        "/api/v1/auth/roles",
		Summary:     "List roles",
		Description: "Returns all registered security roles and their attached permission names.",
		Tags:        []string{"Authorization"},
	}, PermRolesRead), handler.ListRoles)

	huma.Register(api, guard.Protected(huma.Operation{
		OperationID: "get-role",
		Method:      http.MethodGet,
		Path:        "/api/v1/auth/roles/{id}",
		Summary:     "Get role",
		Description: "Retrieves a single security role by ID.",
		Tags:        []string{"Authorization"},
	}, PermRolesRead), handler.GetRole)

	huma.Register(api, guard.Protected(huma.Operation{
		OperationID: "update-role",
		Method:      http.MethodPut,
		Path:        "/api/v1/auth/roles/{id}",
		Summary:     "Update role",
		Description: "Updates a security role's name and description.",
		Tags:        []string{"Authorization"},
	}, PermRolesUpdate), handler.UpdateRole)

	huma.Register(api, guard.Protected(huma.Operation{
		OperationID: "delete-role",
		Method:      http.MethodDelete,
		Path:        "/api/v1/auth/roles/{id}",
		Summary:     "Delete role",
		Description: "Deletes a security role and removes assignments.",
		Tags:        []string{"Authorization"},
	}, PermRolesDelete), handler.DeleteRole)

	huma.Register(api, guard.Protected(huma.Operation{
		OperationID: "add-role-permission",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/roles/{id}/permissions",
		Summary:     "Add permission to role",
		Description: "Attaches a permission string to a security role.",
		Tags:        []string{"Authorization"},
	}, PermRolePermissionsManage), handler.AddRolePermission)

	huma.Register(api, guard.Protected(huma.Operation{
		OperationID: "remove-role-permission",
		Method:      http.MethodDelete,
		Path:        "/api/v1/auth/roles/{id}/permissions/{permission_name}",
		Summary:     "Remove permission from role",
		Description: "Detaches a permission string from a security role.",
		Tags:        []string{"Authorization"},
	}, PermRolePermissionsManage), handler.RemoveRolePermission)

	huma.Register(api, guard.Protected(huma.Operation{
		OperationID: "assign-user-role",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/users/{user_id}/roles",
		Summary:     "Assign role to user",
		Description: "Grants a security role to a user account.",
		Tags:        []string{"Authorization"},
	}, PermUserRolesManage), handler.AssignUserRole)

	huma.Register(api, guard.Protected(huma.Operation{
		OperationID: "remove-user-role",
		Method:      http.MethodDelete,
		Path:        "/api/v1/auth/users/{user_id}/roles/{role_id}",
		Summary:     "Remove role from user",
		Description: "Revokes a security role from a user account.",
		Tags:        []string{"Authorization"},
	}, PermUserRolesManage), handler.RemoveUserRole)
}
