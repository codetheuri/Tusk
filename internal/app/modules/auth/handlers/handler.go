package handlers

import (
	//	"context"
	"net/http"

	 "github.com/codetheuri/todolist/internal/app/modules/auth/services"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/codetheuri/todolist/pkg/validator"
	"github.com/codetheuri/todolist/pkg/validators"
	//"github.com/codetheuri/todolist/pkg/web"
	//"github.com/codetheuri/todolist/internal/app/modules/auth/models"
)

type AuthHandler interface{
	Register(w http.ResponseWriter, r *http.Request)
	// Login(w http.ResponseWriter, r *http.Request)
	// GetUserProfile(w http.ResponseWriter, r *http.Request)
	// ChangePassword(w http.ResponseWriter, r *http.Request)
	// DeleteUser(w http.ResponseWriter, r *http.Request)
	// RestoreUser(w http.ResponseWriter, r *http.Request)
	// Logout(w http.ResponseWriter, r *http.Request)
}
type authHandler struct {
       authServices *services.AuthService
	   log logger.Logger
	   validator *validator.Validator
}

// constructor for AuthHandler
func NewAuthHandler(authServices *services.AuthService, log logger.Logger, validator *validators.Validator) AuthHandler {
	return &authHandler{
		authServices: authServices,
		log: log,
		validator: validator,
	}
}

func (h *authHandler) Register(w http.ResponseWriter, r *http.Request) {
	// Implementation of the Register method will go here
	// This will handle user registration logic
	h.log.Info("Register endpoint hit")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Register endpoint"))
}