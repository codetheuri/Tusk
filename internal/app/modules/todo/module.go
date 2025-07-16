package todo

import (
	todoHandlers "github.com/codetheuri/todolist/internal/app/modules/todo/handlers"
	todoRepositories "github.com/codetheuri/todolist/internal/app/modules/todo/repositories"
	todoServices "github.com/codetheuri/todolist/internal/app/modules/todo/services"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/codetheuri/todolist/pkg/validators"
	"github.com/go-chi/chi"
	"gorm.io/gorm"
)

type Module struct {
	Handlers *todoHandlers.TodoHandler
}

func NewModule(db *gorm.DB, log logger.Logger, validator *validators.Validator) *Module {
	// Initialize the repository
	todoRepo := todoRepositories.NewGormTodoRepository(db, log)

	// Initialize the service
	todoService := todoServices.NewTodoService(todoRepo, validator, log)

	// Initialize the handler
	todoHandler := todoHandlers.NewTodoHandler(todoService, log)

	return &Module{
		Handlers: todoHandler,
	}
}

func (m *Module) RegisterRoutes(r chi.Router) {
	// Register the routes for the todo module
	r.Route("/todos", func(r chi.Router) {
		r.Post("/", m.Handlers.CreateTodo)
		r.Get("/{id}", m.Handlers.GetTodoByID)
		r.Get("/", m.Handlers.GetAllTodos)
		r.Put("/{id}", m.Handlers.UpdateTodo)
		r.Delete("/{id}", m.Handlers.SoftDeleteTodo)
		r.Patch("/{id}/restore", m.Handlers.RestoreTodo)
		r.Delete("/{id}/hard", m.Handlers.HardDeleteTodo)
	})
}
