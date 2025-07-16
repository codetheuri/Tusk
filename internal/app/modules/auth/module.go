package auth

import (
	authHandlers "github.com/codetheuri/todolist/internal/app/modules/auth/handlers"
)

// Module represents the Auth module.
type Module struct {
	Handlers *authHandlers.AuthHandler
}

// // NewModule initializes  Auth module.
// func NewModule(db *gorm.DB, log logger.Logger, validator *validators.Validator) *Module {
// 	repo := authRepositories.NewGormAuthRepository(db, log)
// 	service := authServices.NewAuthService(repo, validator, log)
// 	handler := authHandlers.NewAuthHandler(service, log)

// 	return &Module{
// 		Handlers: handler,
// 	}
// }

// // RegisterRoutes registers the routes for the Auth module.
// func (m *Module) RegisterRoutes(r chi.Router) {
// 	// Register the routes for the auth module
// 	r.Route("/auths", func(r chi.Router) {
// 		r.Post("/", m.Handlers.CreateAuth)
// 		r.Get("/", m.Handlers.GetAllAuths)

// 	})
// }
