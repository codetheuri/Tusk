package auth

import "github.com/codetheuri/tusk/pkg/authz"

// Auth module permission constants to prevent raw string typos across handlers.
const (
	PermUsersRead              = "users.read"
	PermUsersCreate            = "users.create"
	PermUsersUpdate            = "users.update"
	PermUsersDelete            = "users.delete"
	PermRolesRead              = "roles.read"
	PermRolesCreate            = "roles.create"
	PermRolesUpdate            = "roles.update"
	PermRolesDelete            = "roles.delete"
	PermRolePermissionsManage = "roles.permissions.manage"
	PermUserRolesManage        = "users.roles.manage"
	PermPermissionsRead        = "permissions.read"
)

// Permissions exported by the auth module.
var Permissions = []authz.Permission{
	{
		Name:        PermUsersRead,
		Description: "Allows viewing user accounts and profiles",
	},
	{
		Name:        PermUsersCreate,
		Description: "Allows registering and creating new user accounts",
	},
	{
		Name:        PermUsersUpdate,
		Description: "Allows editing existing user accounts",
	},
	{
		Name:        PermUsersDelete,
		Description: "Allows deleting or deactivating user accounts",
	},
	{
		Name:        PermRolesRead,
		Description: "Allows viewing security roles and assigned permissions",
	},
	{
		Name:        PermRolesCreate,
		Description: "Allows creating new security roles",
	},
	{
		Name:        PermRolesUpdate,
		Description: "Allows modifying security role names and descriptions",
	},
	{
		Name:        PermRolesDelete,
		Description: "Allows deleting security roles",
	},
	{
		Name:        PermRolePermissionsManage,
		Description: "Allows attaching or detaching permissions to/from roles",
	},
	{
		Name:        PermUserRolesManage,
		Description: "Allows assigning or revoking roles to/from users",
	},
	{
		Name:        PermPermissionsRead,
		Description: "Allows viewing system permissions",
	},
}

// init registers the module's permissions automatically into the default registry.
func init() {
	authz.Register(Permissions...)
}
