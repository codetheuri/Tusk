package auth

import (
	authHandlers "github.com/codetheuri/todolist/internal/app/modules/auth/handlers"
	authRepositories "github.com/codetheuri/todolist/internal/app/modules/auth/repositories"
	authServices "github.com/codetheuri/todolist/internal/app/modules/auth/services"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/codetheuri/todolist/pkg/validators"
	"github.com/go-chi/chi"
	"gorm.io/gorm"
	"net/http"
)

// Module represents the Auth module.
type Module struct {
	Handlers *authHandlers.AuthHandler

}

// NewModule initializes  Auth module.
func NewModule(db *gorm.DB, log logger.Logger, validator *validators.Validator) *Module {
     repo := authRepositories.NewAuthRepository(db, log)
	 service := authServices.NewAuthService(*repo , validator, log)
	 handler := authHandlers.NewAuthHandler(service, log)

	return &Module{
		Handlers: handler,	
}
}

// RegisterRoutes registers the routes for the Auth module.
func (m *Module) RegisterRoutes(r chi.Router) {
	// Register the routes for the auth module
	r.Route("/auth", func(r chi.Router) {
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Module auth is working!"))
	})
		//r.Post("/", m.Handlers.CreateAuth)
		//r.Get("/", m.Handlers.GetAllAuths)
		
	})
}
