package services

import (
	"time"
	authRepositories "github.com/codetheuri/todolist/internal/app/modules/auth/repositories"
	//"github.com/codetheuri/todolist/internal/app/modules/auth/models"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/codetheuri/todolist/pkg/validators"
)
type AuthService struct {
	UserService  UserService
	TokenService TokenService
}
// service constructor for all services
func NewAuthService(
	repos *authRepositories.AuthRepository,
	validator *validators.Validator,
	jwtSecret string,
	tokenTTL time.Duration,
	log logger.Logger) *AuthService {
	return &AuthService{
	   UserService: NewUserService(repos.UserRepo, validator, log),
	   TokenService: NewTokenService(repos.RevokedTokenRepo, jwtSecret, tokenTTL, log),
	}
}


