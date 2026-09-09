package auth

import (
	"context"
	"net/url"

	"github.com/danielgtaylor/huma/v2"

	"github.com/codetheuri/tusk/v2/pkg/authz"
	"github.com/codetheuri/tusk/v2/pkg/logger"
	"github.com/codetheuri/tusk/v2/pkg/query"
	"github.com/codetheuri/tusk/v2/pkg/response"
)

type Handler struct {
	service *Service
	log     logger.Logger
}

func NewHandler(service *Service, log logger.Logger) *Handler {
	return &Handler{service: service, log: log}
}

// Register creates a new user and identity profile.
func (h *Handler) Register(ctx context.Context, input *RegisterInput) (*AuthOutput, error) {
	if input.Body.Password != input.Body.PasswordConfirm {
		return nil, response.FieldError("Validation failed", "password_confirmation", "Passwords do not match")
	}

	user, err := h.service.Register(ctx, &input.Body)
	if err != nil {
		h.log.Error("Registration failed", err)
		return nil, huma.Error400BadRequest(err.Error(), err)
	}

	// Immediately issue access & refresh tokens on successful registration
	tokens, err := h.service.Login(ctx, &LoginRequest{Login: user.Username, Password: input.Body.Password})
	if err != nil {
		// Return user without token if immediate login fails
		resp := &AuthOutput{}
		resp.Body.Success = true
		resp.Body.Message = "User registered successfully"
		resp.Body.Data.User = user
		return resp, nil
	}

	resp := &AuthOutput{}
	resp.Body.Success = true
	resp.Body.Message = "User registered successfully"
	resp.Body.Data.User = tokens.User
	resp.Body.Data.AccessToken = tokens.AccessToken
	resp.Body.Data.RefreshToken = tokens.RefreshToken
	return resp, nil
}

// Login authenticates a user via single flexible login field (username, email, or phone).
func (h *Handler) Login(ctx context.Context, input *LoginInput) (*AuthOutput, error) {
	tokens, err := h.service.Login(ctx, &input.Body)
	if err != nil {
		return nil, huma.Error401Unauthorized(err.Error(), err)
	}

	resp := &AuthOutput{}
	resp.Body.Success = true
	resp.Body.Message = "User authenticated successfully"
	resp.Body.Data.User = tokens.User
	resp.Body.Data.AccessToken = tokens.AccessToken
	resp.Body.Data.RefreshToken = tokens.RefreshToken
	return resp, nil
}

// RefreshToken rotates refresh tokens and issues a new Access Token.
func (h *Handler) RefreshToken(ctx context.Context, input *RefreshTokenInput) (*AuthOutput, error) {
	tokens, err := h.service.RefreshToken(ctx, input.Body.RefreshToken)
	if err != nil {
		return nil, huma.Error401Unauthorized("Invalid or expired refresh token", err)
	}

	resp := &AuthOutput{}
	resp.Body.Success = true
	resp.Body.Message = "Tokens refreshed successfully"
	resp.Body.Data.User = tokens.User
	resp.Body.Data.AccessToken = tokens.AccessToken
	resp.Body.Data.RefreshToken = tokens.RefreshToken
	return resp, nil
}

// Logout revokes the provided refresh token session.
func (h *Handler) Logout(ctx context.Context, input *RefreshTokenInput) (*MessageOutput, error) {
	if err := h.service.Logout(ctx, input.Body.RefreshToken); err != nil {
		return nil, huma.Error400BadRequest("Failed to revoke session", err)
	}

	resp := &MessageOutput{}
	resp.Body.Success = true
	resp.Body.Message = "Logged out successfully"
	return resp, nil
}

// Me returns the profile and permission list of the authenticated user.
func (h *Handler) Me(ctx context.Context, input *MeInput) (*ProfileOutput, error) {
	sub, ok := authz.SubjectFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Authentication required")
	}
	userID := sub.UserID

	user, perms, err := h.service.GetCurrentUser(ctx, userID)
	if err != nil {
		return nil, huma.Error404NotFound("User profile not found", err)
	}

	resp := &ProfileOutput{}
	resp.Body.Success = true
	resp.Body.Message = "Profile retrieved successfully"
	resp.Body.Data.User = user
	resp.Body.Data.Permissions = perms
	return resp, nil
}

// UpdateProfile updates the personal profile of the authenticated user.
func (h *Handler) UpdateProfile(ctx context.Context, input *UpdateProfileInput) (*ProfileOutput, error) {
	sub, ok := authz.SubjectFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Authentication required")
	}
	userID := sub.UserID

	_, err := h.service.UpdateProfile(ctx, userID, input.Body.FirstName, input.Body.LastName, input.Body.Avatar, input.Body.Bio)
	if err != nil {
		return nil, huma.Error400BadRequest("Failed to update profile", err)
	}

	user, perms, err := h.service.GetCurrentUser(ctx, userID)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to reload profile", err)
	}

	resp := &ProfileOutput{}
	resp.Body.Success = true
	resp.Body.Message = "Profile updated successfully"
	resp.Body.Data.User = user
	resp.Body.Data.Permissions = perms
	return resp, nil
}

// ListUsers handles retrieving paginated user accounts matching search/filter criteria.
func (h *Handler) ListUsers(ctx context.Context, input *ListUsersInput) (*UsersOutput, error) {
	q := query.Query{
		Page:    input.Page,
		PerPage: input.PerPage,
		Search:  input.Search,
		Filters: make(map[string]string),
	}

	if input.Sort != "" {
		sortQuery := query.ParseURLValues(url.Values{"sort": []string{input.Sort}})
		q.Sorts = sortQuery.Sorts
	}

	if input.IsActive != "" {
		q.Filters["is_active"] = input.IsActive
	}

	users, meta, err := h.service.ListUsers(ctx, q)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to retrieve users", err)
	}

	resp := &UsersOutput{}
	resp.Body.Success = true
	resp.Body.Message = "Users retrieved successfully"
	resp.Body.Data.Users = users
	resp.Body.Data.Meta = meta
	return resp, nil
}

// ListPermissions handles returning all registered permissions.
func (h *Handler) ListPermissions(ctx context.Context, input *ListPermissionsInput) (*PermissionsOutput, error) {
	perms, err := h.service.ListPermissions(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to retrieve permissions", err)
	}

	resp := &PermissionsOutput{}
	resp.Body.Success = true
	resp.Body.Message = "Permissions retrieved successfully"
	resp.Body.Data.Permissions = perms
	return resp, nil
}
