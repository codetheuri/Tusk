package services

import (
	"time"
	authRepositories "github.com/codetheuri/todolist/internal/app/auth/repositories"
	//"github.com/codetheuri/todolist/internal/app/modules/auth/models"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/codetheuri/todolist/pkg/validators"
	tokenPkg "github.com/codetheuri/todolist/pkg/auth/token"
)
type AuthService struct {
	UserService  UserService
	TokenService tokenPkg.TokenService
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
	   TokenService: NewJWTService(repos.RevokedTokenRepo, jwtSecret, tokenTTL, log),
	}
}


