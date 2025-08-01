package handlers

import (
	//	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/codetheuri/todolist/internal/app/auth/handlers/dto"
	"github.com/codetheuri/todolist/internal/app/auth/services"
	tokenPkg "github.com/codetheuri/todolist/pkg/auth/token"
	appErrors "github.com/codetheuri/todolist/pkg/errors"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/codetheuri/todolist/pkg/pagination"
	"github.com/codetheuri/todolist/pkg/web"
	"github.com/go-chi/chi"
	"golang.org/x/crypto/bcrypt"

	"github.com/codetheuri/todolist/pkg/validators"
	//"github.com/codetheuri/todolist/internal/app/modules/auth/models"
	"math"
)

type AuthHandler interface {
	Register(w http.ResponseWriter, r *http.Request)
	Login(w http.ResponseWriter, r *http.Request)
	GetUserProfile(w http.ResponseWriter, r *http.Request)
	GetUsers(w http.ResponseWriter, r *http.Request)
	ChangePassword(w http.ResponseWriter, r *http.Request)
	DeleteUser(w http.ResponseWriter, r *http.Request)
	RestoreUser(w http.ResponseWriter, r *http.Request)
	Logout(w http.ResponseWriter, r *http.Request)
}
type authHandler struct {
	authServices *services.AuthService
	log          logger.Logger
	validator    *validators.Validator
}

// constructor for AuthHandler
func NewAuthHandler(authServices *services.AuthService, log logger.Logger, validator *validators.Validator) AuthHandler {
	return &authHandler{
		authServices: authServices,
		log:          log,
		validator:    validator,
	}
}

func (h *authHandler) Register(w http.ResponseWriter, r *http.Request) {
	h.log.Info("Handler: Received registration request")

	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("Handler: Failed to decode registration request", err)
		web.RespondError(w, appErrors.ValidationError("invalid request payload", err, nil), http.StatusBadRequest)
		return
	}
	validationErrors := h.validator.Struct(req)
	if validationErrors != nil {
		h.log.Warn("Handler: Validation failed for registration request", "errors", validationErrors)
		web.RespondError(w, appErrors.ValidationError("validation failed", nil, validationErrors), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	user, err := h.authServices.UserService.RegisterUser(ctx, req.Email, req.Password, req.Role)
	if err != nil {
		h.log.Error("Handler: Failed to register user through service", err, "email", req.Email)
		h.handleAppError(w, err, "user registration")
		return
	}

	tokenString, err := h.authServices.TokenService.GenerateToken(fmt.Sprintf("%d", user.ID), user.Role)
	if err != nil {
		h.log.Error("Handler: Failed to generate auth token after registration", err, "userID", user.ID)
		web.RespondError(w, appErrors.InternalServerError("failed to generate authentication token", err), http.StatusInternalServerError)
		return
	}
	resp := dto.AuthResponse{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		Token:  tokenString,

		ExpiresAt: h.authServices.TokenService.GetTokenTTL().Unix(), // Access token TTL from TokenService
	}

	h.log.Info("Handler: User registered and token generated", "userID", user.ID)

	web.RespondData(w, http.StatusCreated, resp, "User registered successfully", web.WithSuccessType("toast"))

}

func (h *authHandler) Login(w http.ResponseWriter, r *http.Request) {
	h.log.Info("Handler: Received login request")

	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("Handler: Failed to decode login request", err)
		web.RespondError(w, appErrors.ValidationError("invalid request payload", err, nil), http.StatusBadRequest)
		return
	}

	validationErrors := h.validator.Struct(req)
	if validationErrors != nil {
		h.log.Warn("Handler: Validation failed for login request", "errors", validationErrors)
		web.RespondError(w, appErrors.ValidationError("validation failed", nil, validationErrors), http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// 1. Get user by email
	user, err := h.authServices.UserService.GetUserByEmail(ctx, req.Email)
	if err != nil {
		h.log.Error("Handler: Failed to get user by email during login", err, "email", req.Email)

		var authErr appErrors.AppError
		if errors.As(err, &authErr) && authErr.Code() == "AUTH_ERROR" {
			web.RespondError(w, authErr, http.StatusUnauthorized,
				web.WithAlertifyType("toast"),
				web.WithAlertifyTheme("danger"),
				web.WithAlertifyMessage("Invalid email or password"),
			)
		} else {
			h.handleAppError(w, err, "user login")
		}
		return
	}

	// 2. Compare passwords
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		h.log.Warn("Handler: Invalid password attempt for user", "email", req.Email, "error", err)
		// web.RespondError(w, appErrors.AuthError("invalid credentials", nil), http.StatusUnauthorized)
		web.RespondError(w, appErrors.AuthError("Invalid credentials", nil), http.StatusUnauthorized,
			web.WithAlertifyType("toast"),
			web.WithAlertifyTheme("danger"),
			web.WithAlertifyMessage("Invalid email or password"),
		)

		return
	}

	// 3. Generate Auth Token
	tokenString, err := h.authServices.TokenService.GenerateToken(fmt.Sprintf("%d", user.ID), user.Role)
	if err != nil {
		h.log.Error("Handler: Failed to generate auth token after successful login", err, "userID", user.ID)
		web.RespondError(w, appErrors.InternalServerError("failed to generate authentication token", err), http.StatusInternalServerError)
		return
	}

	resp := dto.AuthResponse{
		UserID:    user.ID,
		Email:     user.Email,
		Role:      user.Role,
		Token:     tokenString,
		ExpiresAt: h.authServices.TokenService.GetTokenTTL().Unix(), // Access token TTL from TokenService
	}

	h.log.Info("Handler: User logged in successfully", "userID", user.ID)
	// extraSlice := map[string]string{"message":"access granted", "theme":"primary", "type":"toast"}

	web.RespondData(w, http.StatusOK, resp, "access granted",
		// web.WithMetadata(extraSlice),
		web.WithSuccessType("toast"),

		// web.WithSuccessMessage("Login successful"),
		// web.WithSuccessTheme("primary"),
	)

}

func (h *authHandler) GetUserProfile(w http.ResponseWriter, r *http.Request) {
	h.log.Info("Handler: Received GetUserProfile request")

	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		h.log.Warn("Handler: Invalid user ID format in URL", err, "id", userIDStr)
		web.RespondError(w, appErrors.ValidationError("invalid user ID format", nil, nil), http.StatusBadRequest)
		return
	}
	// Bounds check: ensure userID fits in uint
	if userID > uint64(math.MaxUint) {
		h.log.Warn("Handler: User ID out of range for uint", "id", userIDStr)
		web.RespondError(w, appErrors.ValidationError("user ID out of range", nil, nil), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	// Bounds check: ensure userID fits in uint
	if userID > uint64(math.MaxUint) {
		h.log.Warn("Handler: User ID out of range for uint", "id", userIDStr)
		web.RespondError(w, appErrors.ValidationError("user ID out of range", nil, nil), http.StatusBadRequest)
		return
	}

	user, err := h.authServices.UserService.GetUserByID(ctx, uint(userID))
	if err != nil {
		h.log.Error("Handler: Failed to get user profile through service", err, "userID", userID)
		h.handleAppError(w, err, "get user profile")
		return
	}

	resp := dto.GetUserProfileResponse{
		UserID:    user.ID,
		Email:     user.Email,
		CreatedAt: &user.CreatedAt, // Ensure CreatedAt is included

		// Role:   user.Role,
	}

	h.log.Info("Handler: User profile retrieved successfully", "userID", user.ID)

	web.RespondData(w, http.StatusOK, resp, "User profile retrieved successfully", web.WithoutSuccess())
}

func (h *authHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	h.log.Info("Handler: Received ChangePassword request")
	ctxUserID, ok := tokenPkg.GetUserIDFromContext(r.Context())
	if !ok {
		h.log.Warn("Handler: UserID not found in context for ChangePassword (middleware error or missing)")
		web.RespondError(w, appErrors.AuthError("authentication context missing", nil), http.StatusUnauthorized)
		return
	}
	//for admins
	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		h.log.Warn("Handler: Invalid user ID format in URL for change password", err, "id", userIDStr)
		web.RespondError(w, appErrors.ValidationError("invalid user ID format", nil, nil), http.StatusBadRequest)
		return
	}

	var req dto.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("Handler: Failed to decode change password request", err)
		web.RespondError(w, appErrors.ValidationError("invalid request payload", err, nil), http.StatusBadRequest)
		return
	}

	validationErrors := h.validator.Struct(req)
	if validationErrors != nil {
		h.log.Warn("Handler: Validation failed for change password request", "errors", validationErrors)
		web.RespondError(w, appErrors.ValidationError("validation failed", nil, validationErrors), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	err = h.authServices.UserService.ChangePassword(ctx, ctxUserID, req.OldPassword, req.NewPassword)
	if err != nil {
		h.log.Error("Handler: Failed to change password through service", err, "userID", userID)
		h.handleAppError(w, err, "change password")
		return
	}

	h.log.Info("Handler: Password changed successfully", "userID", userID)
	// web.RespondJSON(w, http.StatusOK, dto.SuccessResponse{Message: "Password changed successfully"})
	web.RespondMessage(w, http.StatusOK, "Password changed successfully", "success", "alert")
}

func (h *authHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	h.log.Info("Handler: Received DeleteUser request")

	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		h.log.Warn("Handler: Invalid user ID format for deletion", err, "id", userIDStr)
		web.RespondError(w, appErrors.ValidationError("invalid user ID format", nil, nil), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if userID > math.MaxUint {
		h.log.Warn("Handler: User ID out of range for uint type", "userID", userID)
		web.RespondError(w, appErrors.ValidationError("user ID out of range", nil, nil), http.StatusBadRequest)
		return
	}
	err = h.authServices.UserService.DeleteUser(ctx, uint(userID))
	if err != nil {
		h.log.Error("Handler: Failed to delete user through service", err, "userID", userID)
		h.handleAppError(w, err, "delete user")
	}

	h.log.Info("Handler: User soft-deleted successfully", "userID", userID)
	web.RespondMessage(w, http.StatusNoContent, "User soft-deleted successfully", "success", "toast")
	// web.RespondJSON(w, http.StatusNoContent, dto.SuccessResponse{Message: "user deleted successfully"}) // 204 No Content for successful deletion

}

func (h *authHandler) RestoreUser(w http.ResponseWriter, r *http.Request) {
	h.log.Info("Handler: Received RestoreUser request")

	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		h.log.Warn("Handler: Invalid user ID format for restore", err, "id", userIDStr)
		web.RespondError(w, appErrors.ValidationError("invalid user ID format", nil, nil), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if userID > math.MaxUint {
		h.log.Warn("Handler: User ID out of range for uint type", "userID", userID)
		web.RespondError(w, appErrors.ValidationError("user ID out of range", nil, nil), http.StatusBadRequest)
		return
	}
	err = h.authServices.UserService.RestoreUser(ctx, uint(userID))
	if err != nil {
		h.log.Error("Handler: Failed to restore user through service", err, "userID", userID)
		h.handleAppError(w, err, "restore user")
		return
	}

	h.log.Info("Handler: User restored successfully", "userID", userID)
	web.RespondMessage(w, http.StatusOK, "User restored successfully", "success", "toast")

}

func (h *authHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.log.Info("Handler: Received Logout request")

	jti, jtiOK := tokenPkg.GetJTIFromContext(r.Context())
	expiresAt, expOK := tokenPkg.GetExpiresAtFromContext(r.Context())

	if !jtiOK || !expOK {
		h.log.Warn("Handler: Logout request missing JTI or ExpiresAt in context. Ensure Authenticator middleware is active and correctly extracts these.")
		web.RespondError(w, appErrors.AuthError("authentication context missing for logout", nil), http.StatusUnauthorized)
		return
	}
	ctx := r.Context()
	err := h.authServices.TokenService.RevokeToken(ctx, jti, expiresAt)
	if err != nil {
		h.log.Error("Handler: Failed to revoke token through service", err, "jti", jti)
		h.handleAppError(w, err, "logout")
		return
	}

	h.log.Info("Handler: User logged out successfully (token revoked)", "jti", jti)
	web.RespondMessage(w, http.StatusOK, "Logged out successfully", "success", "toast")

}
func (h *authHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	h.log.Debug("Handler: Received Get users request")
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		page = pagination.DefaultPage
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = pagination.DefaultLimit
	}
	ctx := r.Context()

	pParams := pagination.NewPaginationParams(page, limit)
	users, totalCount, err := h.authServices.UserService.GetUsers(ctx, pParams.Offset(), pParams.Limit)
	if err != nil {
		h.log.Error("Handler: Service call failed for GetUsers", err)
		h.handleAppError(w, err, "get users")
		return
	}
	userResponses := make([]dto.GetUserProfileResponse, len(users))
	for i, user := range users {
		userResponses[i] = dto.GetUserProfileResponse{
			UserID: user.ID,
			Email:  user.Email,
			// Role:      user.Role,
			CreatedAt: &user.CreatedAt,
		}
	}
	metadata := pagination.NewPaginationmetadata(pParams.Page, pParams.Limit, totalCount)

	web.RespondListData(w, http.StatusOK, userResponses, metadata)
}
func (h *authHandler) handleAppError(w http.ResponseWriter, err error, action string) {
	var appErr appErrors.AppError
	if errors.As(err, &appErr) {
		h.log.Error(fmt.Sprintf("Handler: Application error during %s", action), err, "code", appErr.Code())
		switch appErr.Code() {
		case "AUTH_ERROR":
			web.RespondError(w, appErr, http.StatusUnauthorized)
		case "NOT_FOUND":
			web.RespondError(w, appErr, http.StatusNotFound)
		case "VALIDATION_ERROR": // Keep this, it handles generic validation failures
			web.RespondError(w, appErr, http.StatusBadRequest) // Often 400 or 422 for validation
		case "DATABASE_ERROR":
			web.RespondError(w, appErr, http.StatusInternalServerError)
		case "INTERNAL_SERVER_ERROR":
			web.RespondError(w, appErr, http.StatusInternalServerError)
		case "CONFLICT_ERROR": // <--- ADD THIS CASE
			web.RespondError(w, appErr, http.StatusConflict) // HTTP 409 Conflict
		case "FORBIDDEN": // Ensure this is also handled
			web.RespondError(w, appErr, http.StatusForbidden)
		case "UNAUTHORIZED": // Differentiate from AUTH_ERROR if needed, though often similar
			web.RespondError(w, appErr, http.StatusUnauthorized)
		default:
			web.RespondError(w, appErrors.InternalServerError(
				fmt.Sprintf("an unexpected application error occurred with code %s", appErr.Code()), appErr), http.StatusInternalServerError)
		}
	} else {
		h.log.Error(fmt.Sprintf("Handler: Unknown error during %s", action), err)
		web.RespondError(w, appErrors.InternalServerError("an unknown error occurred", err), http.StatusInternalServerError)
	}
}
