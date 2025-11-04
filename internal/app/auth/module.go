package auth

import (
	"context"

	"github.com/codetheuri/todolist/config"
	authHandlers "github.com/codetheuri/todolist/internal/app/auth/handlers"
	"github.com/codetheuri/todolist/internal/app/auth/handlers/dto"
	authRepositories "github.com/codetheuri/todolist/internal/app/auth/repositories"
	authServices "github.com/codetheuri/todolist/internal/app/auth/services"
	router "github.com/codetheuri/todolist/internal/app/routers"
	tokenPkg "github.com/codetheuri/todolist/pkg/auth/token"
	appErrors "github.com/codetheuri/todolist/pkg/errors"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/codetheuri/todolist/pkg/tonic"
	"github.com/codetheuri/todolist/pkg/validators"
	"gorm.io/gorm"
)

// @title Tusk Authentication API
// @version 1.0.0
// @description This service manages user identity, authentication, and authorization for the Tusk application.
// @host localhost:8081
// @BasePath /api
// @schemes http
// @contact.name Joseph Theuri
// @contact.url http://www.swagger.io/support
// @contact.email support@tuskapp.com
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the JWT token.

type Module struct {
	Handler      *authHandlers.AuthHandlers
	log          logger.Logger
	TokenService tokenPkg.TokenService
	validator    *validators.Validator
}

// NewModule initializes  Auth module.
func NewModule(db *gorm.DB, log logger.Logger, validator *validators.Validator, cfg *config.Config) *Module {
	repos := authRepositories.NewAuthRepository(db, log)

	jwtSecret := cfg.JWTSecret
	tokenTTL := cfg.AccessTokenTTL

	TokenService := authServices.NewJWTService(repos.RevokedTokenRepo, jwtSecret, tokenTTL, log)
	services := authServices.NewAuthService(repos, validator, jwtSecret, tokenTTL, log)
	handler := authHandlers.NewAuthHandler(services, log, validator)

	return &Module{
		Handler:      handler,
		TokenService: TokenService,
		log:          log,
		validator:    validator,
	}
}

type localHandlerAdapterFunc func(ctx context.Context, input interface{}) (interface{}, error)

func createTonicAdapterBridge[T any](
	handlerMethod func(ctx context.Context, req *T) (*tonic.Response, error),
) tonic.HandlerFunc {
	// Returns an explicitly typed tonic.HandlerFunc
	return func(ctx context.Context, input interface{}) (*tonic.Response, error) {
		// 1. Assert the input type safely back to the specific DTO pointer
		req, ok := input.(*T)
		if !ok {
			// Safety fallback: This should not happen if Adapter is called correctly
			return nil, appErrors.InternalServerError("internal type assertion failure", nil)
		}

		// 2. Call the original, strongly-typed handler method
		return handlerMethod(ctx, req)
	}
}

// RegisterRoutes registers the routes for the Auth module.
func (m *Module) RegisterRoutes(r router.Router) {
	m.log.Info("Registering Auth module routes...")
	v := m.validator
	h := m.Handler
	r.Group(func(r router.Router) {
		registerHandlerFunc := createTonicAdapterBridge(h.Register)
		r.Post("/auth/register", tonic.Adapter(registerHandlerFunc, dto.RegisterRequest{}, v))
		// registerHandler := buildAdapterFunction[*dto.RegisterRequest](h, h.Register)
		// r.Post("/auth/register", tonic.Adapter(registerHandler, dto.RegisterRequest{}, v))
		// r.Post("/auth/register", tonic.Adapter(registerAdapterFunc, dto.RegisterRequest{}, v))
		// r.Post("/auth/login", m.Handler.Login)
	})

	// Authenticated routes (will need middleware later)

	// r.Group(func(r router.Router) {
	// 	r.Use(middleware.Authenticator(m.TokenService,m.log))
	// 	r.Get("/auth/profile/{id}", m.Handler.GetUserProfile)
	// 	r.Put("/auth/users/{id}/change-password", m.Handler.ChangePassword)
	// 	r.Delete("/auth/users/{id}", m.Handler.DeleteUser)
	// 	r.Put("/auth/users/{id}/restore", m.Handler.RestoreUser)
	// 	r.Post("/auth/logout", m.Handler.Logout)
	// 	r.Get("/auth/users", m.Handler.GetUsers)
	// })

	m.log.Info("Auth module routes registered.")
}
