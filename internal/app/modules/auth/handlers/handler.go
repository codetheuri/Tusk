package handlers

import (
	//	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/codetheuri/todolist/internal/app/modules/auth/handlers/dto"
	"github.com/codetheuri/todolist/internal/app/modules/auth/services"
	appErrors "github.com/codetheuri/todolist/pkg/errors"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/codetheuri/todolist/pkg/web"
	"github.com/go-chi/chi"
	"golang.org/x/crypto/bcrypt"

	"github.com/codetheuri/todolist/pkg/validators"
	//"github.com/codetheuri/todolist/internal/app/modules/auth/models"
)

type AuthHandler interface {
	Register(w http.ResponseWriter, r *http.Request)
	Login(w http.ResponseWriter, r *http.Request)
	GetUserProfile(w http.ResponseWriter, r *http.Request)
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
		var appErr appErrors.AppError
		if errors.As(err, &appErr) {
			switch appErr.Code() {
			case "AUTH_ERROR":
				web.RespondError(w, appErr, http.StatusConflict)
			case "DATABASE_ERROR":
				web.RespondError(w, appErr, http.StatusInternalServerError)
			case "INTERNAL_SERVER_ERROR":
				web.RespondError(w, appErr, http.StatusInternalServerError)
			case "VALIDATION_ERROR":
				web.RespondError(w, appErr, http.StatusBadRequest)
			default:
				web.RespondError(w, appErrors.InternalServerError("an unexpected error occurred during registration", err), http.StatusInternalServerError)
			}
		} else {
			web.RespondError(w, appErrors.InternalServerError("an unknown error occurred during registration", err), http.StatusInternalServerError)
		}
		return
	}

	tokenString, err := h.authServices.TokenService.GenerateAuthTokens(ctx, user)
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

		ExpiresAt: h.authServices.TokenService.GetTokenTTL().Unix(),
	}

	h.log.Info("Handler: User registered and token generated", "userID", user.ID)
	web.RespondJSON(w, http.StatusCreated, resp)
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
		var appErr appErrors.AppError
		if errors.As(err, &appErr) {
			switch appErr.Code() {
			case "NOT_FOUND":
				web.RespondError(w, appErrors.AuthError("invalid credentials", nil), http.StatusUnauthorized)
			case "DATABASE_ERROR":
				web.RespondError(w, appErr, http.StatusInternalServerError)
			default:
				web.RespondError(w, appErrors.InternalServerError("an unexpected error occurred during login", err), http.StatusInternalServerError)
			}
		} else {
			web.RespondError(w, appErrors.InternalServerError("an unknown error occurred during login", err), http.StatusInternalServerError)
		}
		return
	}

	// 2. Compare passwords
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		h.log.Warn("Handler: Invalid password attempt for user", "email", req.Email)
		web.RespondError(w, appErrors.AuthError("invalid credentials", nil), http.StatusUnauthorized)
		return
	}

	// 3. Generate Auth Token
	tokenString, err := h.authServices.TokenService.GenerateAuthTokens(ctx, user)
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
	web.RespondJSON(w, http.StatusOK, resp)
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

	ctx := r.Context()
	user, err := h.authServices.UserService.GetUserByID(ctx, uint(userID))
	if err != nil {
		h.log.Error("Handler: Failed to get user profile through service", err, "userID", userID)
		var appErr appErrors.AppError
		if errors.As(err, &appErr) {
			switch appErr.Code() {
			case "NOT_FOUND":
				web.RespondError(w, appErr, http.StatusNotFound)
			case "DATABASE_ERROR":
				web.RespondError(w, appErr, http.StatusInternalServerError)
			default:
				web.RespondError(w, appErrors.InternalServerError("an unexpected error occurred", err), http.StatusInternalServerError)
			}
		} else {
			web.RespondError(w, appErrors.InternalServerError("an unknown error occurred", err), http.StatusInternalServerError)
		}
		return
	}

	resp := dto.GetUserProfileResponse{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
	}

	h.log.Info("Handler: User profile retrieved successfully", "userID", user.ID)
	web.RespondJSON(w, http.StatusOK, resp)
}

func (h *authHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	h.log.Info("Handler: Received ChangePassword request")

	// User ID should come from authenticated context, not URL/body for security.
	userIDStr := chi.URLParam(r, "id") // Assuming /auth/users/{id}/change-password
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
	err = h.authServices.UserService.ChangePassword(ctx, uint(userID), req.OldPassword, req.NewPassword)
	if err != nil {
		h.log.Error("Handler: Failed to change password through service", err, "userID", userID)
		var appErr appErrors.AppError
		if errors.As(err, &appErr) {
			switch appErr.Code() {
			case "NOT_FOUND":
				web.RespondError(w, appErr, http.StatusNotFound)
			case "AUTH_ERROR":
				web.RespondError(w, appErr, http.StatusUnauthorized)
			case "DATABASE_ERROR":
				web.RespondError(w, appErr, http.StatusInternalServerError)
			case "INTERNAL_SERVER_ERROR":
				web.RespondError(w, appErr, http.StatusInternalServerError)
			case "VALIDATION_ERROR":
				web.RespondError(w, appErr, http.StatusBadRequest)
			default:
				web.RespondError(w, appErrors.InternalServerError("an unexpected error occurred", err), http.StatusInternalServerError)
			}
		} else {
			web.RespondError(w, appErrors.InternalServerError("an unknown error occurred", err), http.StatusInternalServerError)
		}
		return
	}

	h.log.Info("Handler: Password changed successfully", "userID", userID)
	web.RespondJSON(w, http.StatusOK, dto.SuccessResponse{Message: "Password changed successfully"})
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
	err = h.authServices.UserService.DeleteUser(ctx, uint(userID))
	if err != nil {
		h.log.Error("Handler: Failed to delete user through service", err, "userID", userID)
		var appErr appErrors.AppError
		if errors.As(err, &appErr) {
			switch appErr.Code() {
			case "NOT_FOUND":
				web.RespondError(w, appErr, http.StatusNotFound)
			case "DATABASE_ERROR":
				web.RespondError(w, appErr, http.StatusInternalServerError)
			case "INTERNAL_SERVER_ERROR":
				web.RespondError(w, appErr, http.StatusInternalServerError)
			default:
				web.RespondError(w, appErrors.InternalServerError("an unexpected error occurred during user deletion", err), http.StatusInternalServerError)
			}
		} else {
			web.RespondError(w, appErrors.InternalServerError("an unknown error occurred during user deletion", err), http.StatusInternalServerError)
		}
		return
	}

	h.log.Info("Handler: User soft-deleted successfully", "userID", userID)
	web.RespondJSON(w, http.StatusNoContent, dto.SuccessResponse{Message: "user deleted successfully"}) // 204 No Content for successful deletion
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
	err = h.authServices.UserService.RestoreUser(ctx, uint(userID))
	if err != nil {
		h.log.Error("Handler: Failed to restore user through service", err, "userID", userID)
		var appErr appErrors.AppError
		if errors.As(err, &appErr) {
			switch appErr.Code() {
			case "NOT_FOUND":
				web.RespondError(w, appErr, http.StatusNotFound)
			case "DATABASE_ERROR":
				web.RespondError(w, appErr, http.StatusInternalServerError)
			case "INTERNAL_SERVER_ERROR":
				web.RespondError(w, appErr, http.StatusInternalServerError)
			default:
				web.RespondError(w, appErrors.InternalServerError("an unexpected error occurred during user restore", err), http.StatusInternalServerError)
			}
		} else {
			web.RespondError(w, appErrors.InternalServerError("an unknown error occurred during user restore", err), http.StatusInternalServerError)
		}
		return
	}

	h.log.Info("Handler: User restored successfully", "userID", userID)
	web.RespondJSON(w, http.StatusOK, dto.SuccessResponse{Message: "User restored successfully"})
}


func (h *authHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.log.Info("Handler: Received Logout request")

	// In a real application, middleware would extract the JWT token (JTI and ExpiresAt)
	// from the Authorization header and put it into the request context.
	// For now, this is a placeholder using context values, assuming middleware is coming.
	jti := r.Context().Value("jti")
	expiresAt := r.Context().Value("exp")

	if jti == nil || expiresAt == nil {
		h.log.Warn("Handler: Logout request missing JTI or ExpiresAt in context (middleware not providing)")
		web.RespondError(w, appErrors.AuthError("invalid token for logout or token data missing", nil), http.StatusUnauthorized)
		return
	}

	// Type assertion for context values
	jtiStr, ok := jti.(string)
	if !ok {
		h.log.Error("Handler: JTI in context is not a string, type assertion failed", nil, "jti_type", fmt.Sprintf("%T", jti))
		web.RespondError(w, appErrors.InternalServerError("internal error processing JTI", nil), http.StatusInternalServerError)
		return
	}

	expiresAtTime, ok := expiresAt.(time.Time)
	if !ok {
		h.log.Error("Handler: ExpiresAt in context is not time.Time, type assertion failed", nil, "exp_type", fmt.Sprintf("%T", expiresAt))
		// Attempt to convert from int64 if it's stored as Unix timestamp
		expInt64, isInt := expiresAt.(int64)
		if isInt {
			expiresAtTime = time.Unix(expInt64, 0)
		} else {
			web.RespondError(w, appErrors.InternalServerError("internal error processing ExpiresAt", nil), http.StatusInternalServerError)
			return
		}
	}

	ctx := r.Context()
	err := h.authServices.TokenService.RevokeToken(ctx, jtiStr, expiresAtTime)
	if err != nil {
		h.log.Error("Handler: Failed to revoke token through service", err, "jti", jtiStr)
		var appErr appErrors.AppError
		if errors.As(err, &appErr) {
			switch appErr.Code() {
			case "DATABASE_ERROR":
				web.RespondError(w, appErr, http.StatusInternalServerError)
			case "INTERNAL_SERVER_ERROR":
				web.RespondError(w, appErr, http.StatusInternalServerError)
			default:
				web.RespondError(w, appErrors.InternalServerError("an unexpected error occurred during logout", err), http.StatusInternalServerError)
			}
		} else {
			web.RespondError(w, appErrors.InternalServerError("an unknown error occurred during logout", err), http.StatusInternalServerError)
		}
		return
	}

	h.log.Info("Handler: User logged out successfully (token revoked)", "jti", jtiStr)
	web.RespondJSON(w, http.StatusOK, dto.SuccessResponse{Message: "Logged out successfully"})
}
