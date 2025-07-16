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
	Handers *todoHandlers.TodoHandler
}

func NewModule(db *gorm.DB, log logger.Logger, validator *validators.Validator) *Module {
	// Initialize the repository
	todoRepo := todoRepositories.NewGormTodoRepository(db, log)

	// Initialize the service
	todoService := todoServices.NewTodoService(todoRepo, validator, log)

	// Initialize the handler
	todoHandler := todoHandlers.NewTodoHandler(todoService, log)

	return &Module{
		Handers: todoHandler,
	}
}

func (m *Module) RegisterRoutes(r chi.Router) {
	// Register the routes for the todo module
	r.Route("/todos", func(r chi.Router) {
		r.Post("/", m.Handers.CreateTodo)
		r.Get("/{id}", m.Handers.GetTodoByID)
		r.Get("/", m.Handers.GetAllTodos)
		r.Put("/{id}", m.Handers.UpdateTodo)
		r.Delete("/{id}", m.Handers.SoftDeleteTodo)
		r.Patch("/{id}/restore", m.Handers.RestoreTodo)
		r.Delete("/{id}/hard", m.Handers.HardDeleteTodo)
	})
}
