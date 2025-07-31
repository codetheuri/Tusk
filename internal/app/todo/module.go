package todo

import (
	todoHandlers "github.com/codetheuri/todolist/internal/app/todo/handlers"
	todoRepositories "github.com/codetheuri/todolist/internal/app/todo/repositories"
	todoServices "github.com/codetheuri/todolist/internal/app/todo/services"
	tokenPkg "github.com/codetheuri/todolist/pkg/auth/token"
	"github.com/codetheuri/todolist/pkg/middleware"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/codetheuri/todolist/pkg/validators"
	"github.com/go-chi/chi"
	"gorm.io/gorm"
)

type Module struct {
	Handlers *todoHandlers.TodoHandler
	log      logger.Logger
	TokenService tokenPkg.TokenService
}

func NewModule(db *gorm.DB, log logger.Logger, validator *validators.Validator, tokenService tokenPkg.TokenService) *Module {
	// Initialize the repository
	todoRepo := todoRepositories.NewGormTodoRepository(db, log)

	// Initialize the service
	todoService := todoServices.NewTodoService(todoRepo, validator, log)

	// Initialize the handler
	todoHandler := todoHandlers.NewTodoHandler(todoService, log)

	return &Module{
		Handlers: todoHandler,
		log: 	log,
		TokenService: tokenService,
	}
}

func (m *Module) RegisterRoutes(r chi.Router) {
	// Register the routes for the todo module
	r.Route("/todos", func(r chi.Router) {
		r.Use(middleware.Authenticator(m.TokenService, m.log)) // Apply authentication middleware
		r.Post("/", m.Handlers.CreateTodo)
		r.Get("/all", m.Handlers.GetAllIncludingDeleted)
		r.Get("/{id}", m.Handlers.GetTodoByID)
		r.Get("/", m.Handlers.GetAllTodos)
		r.Put("/{id}", m.Handlers.UpdateTodo)
		r.Delete("/{id}", m.Handlers.SoftDeleteTodo)
		r.Patch("/{id}/restore", m.Handlers.RestoreTodo)
		r.Delete("/{id}/hard", m.Handlers.HardDeleteTodo)
	})
}
