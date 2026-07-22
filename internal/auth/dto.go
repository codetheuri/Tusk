package auth

import (
	"github.com/codetheuri/tusk/pkg/authz"
	"github.com/codetheuri/tusk/pkg/query"
	"github.com/codetheuri/tusk/pkg/response"
)

// -------------------------------------------------------------
// AUTHENTICATION & IDENTITY DTOs
// -------------------------------------------------------------

type RegisterRequest struct {
	Username        string  `json:"username" minLength:"3" doc:"The user's chosen username"`
	Email           string  `json:"email" format:"email" doc:"The user's email address"`
	Phone           *string `json:"phone,omitempty" doc:"Optional phone number"`
	Password        string  `json:"password" minLength:"8" doc:"The user's secure password"`
	PasswordConfirm string  `json:"password_confirmation" minLength:"8" doc:"Must match password"`
	FirstName       string  `json:"first_name,omitempty" doc:"First name"`
	LastName        string  `json:"last_name,omitempty" doc:"Last name"`
}

type LoginRequest struct {
	Login    string `json:"login" doc:"Username, Email address, or Phone number"`
	Password string `json:"password" doc:"User password"`
}

type RegisterInput struct {
	Body RegisterRequest
}

type LoginInput struct {
	Body LoginRequest
}

type RefreshTokenInput struct {
	Body struct {
		RefreshToken string `json:"refresh_token" doc:"Valid Refresh Token"`
	}
}

type UpdateProfileInput struct {
	Body struct {
		FirstName string `json:"first_name" doc:"Updated first name"`
		LastName  string `json:"last_name" doc:"Updated last name"`
		Avatar    string `json:"avatar" doc:"Updated avatar URL"`
		Bio       string `json:"bio" doc:"Updated bio"`
	}
}

type AuthResultData struct {
	User         *User  `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type AuthOutput struct {
	Body response.Data[AuthResultData]
}

type MeInput struct{}

type ProfileData struct {
	User        *User    `json:"user"`
	Permissions []string `json:"permissions"`
}

type ProfileOutput struct {
	Body response.Data[ProfileData]
}

type ListUsersInput struct {
	Page     int    `query:"page" doc:"Page number (default 1)"`
	PerPage  int    `query:"per_page" doc:"Items per page (default 20)"`
	Search   string `query:"search" doc:"Search across username, email, and phone"`
	Sort     string `query:"sort" doc:"Sort field e.g. -created_at or username"`
	IsActive string `query:"is_active" doc:"Filter by active status (true/false)"`
}

type UsersData struct {
	Users []User     `json:"users"`
	Meta  query.Meta `json:"meta"`
}

type UsersOutput struct {
	Body response.Data[UsersData]
}

type ListPermissionsInput struct{}

type PermissionsData struct {
	Permissions []authz.Permission `json:"permissions"`
}

type PermissionsOutput struct {
	Body response.Data[PermissionsData]
}

// -------------------------------------------------------------
// ROLE & PERMISSION DTOs
// -------------------------------------------------------------

type CreateRoleInput struct {
	Body struct {
		Name        string `json:"name" minLength:"2" doc:"Name of the role (e.g., admin, editor)"`
		Description string `json:"description" doc:"Optional description of role capabilities"`
	}
}

type UpdateRoleInput struct {
	ID   uint `path:"id" doc:"Role ID"`
	Body struct {
		Name        *string `json:"name,omitempty" minLength:"2" doc:"Updated name of the role"`
		Description *string `json:"description,omitempty" doc:"Updated description"`
	}
}

type RoleIDInput struct {
	ID uint `path:"id" doc:"Role ID"`
}

type RoleData struct {
	Role *Role `json:"role"`
}

type RoleOutput struct {
	Body response.Data[RoleData]
}

type ListRolesInput struct{}

type RolesData struct {
	Roles []Role `json:"roles"`
}

type RolesOutput struct {
	Body response.Data[RolesData]
}

type AddRolePermissionInput struct {
	ID   uint `path:"id" doc:"Role ID"`
	Body struct {
		PermissionName string `json:"permission_name" doc:"Permission string to attach (e.g., users.read)"`
	}
}

type RemoveRolePermissionInput struct {
	ID             uint   `path:"id" doc:"Role ID"`
	PermissionName string `path:"permission_name" doc:"Permission string to remove"`
}

type AssignUserRoleInput struct {
	UserID uint `path:"user_id" doc:"User ID"`
	Body   struct {
		RoleID uint `json:"role_id" doc:"Role ID to assign"`
	}
}

type RemoveUserRoleInput struct {
	UserID uint `path:"user_id" doc:"User ID"`
	RoleID uint `path:"role_id" doc:"Role ID to revoke"`
}

type MessageOutput struct {
	Body response.Data[struct{}]
}
