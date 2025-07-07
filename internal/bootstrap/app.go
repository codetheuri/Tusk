package bootstrap

import (
	"fmt"
	"net/http"

	"github.com/codetheuri/todolist/config"
	"github.com/codetheuri/todolist/internal/app/handlers"
	"github.com/codetheuri/todolist/internal/app/models"
	"github.com/codetheuri/todolist/internal/app/repositories"
	"github.com/codetheuri/todolist/internal/app/services"
	"github.com/codetheuri/todolist/internal/platform/database"
	appErrors "github.com/codetheuri/todolist/pkg/errors"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/codetheuri/todolist/pkg/validators"
	"github.com/codetheuri/todolist/pkg/middleware" 
	"github.com/codetheuri/todolist/pkg/web"
	// "github.com/codetheuri/todolist/pkg/validators"
)

// initiliazes and start the application
func Run(cfg *config.Config, log logger.Logger) error {
	//db
	db, err := database.NewGoRMDB(cfg, log)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Info("Running migrations...")
	// err = db.Migrator().CreateTable(&models.Todo{})
	// if err != nil {
	// 	return fmt.Errorf("failed to create table: %w", err)
	// }
	if err := db.AutoMigrate(&models.Todo{}); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	log.Info("Migrations completed successfully")

	//initialize the router

	//initilialize app components
	appValidator := validators.NewValidator()

	//initialize the repositories
	todoRepo := repositories.NewGormTodoRepository(db, log)
	//initilliaze services
	todoService := services.NewTodoService(todoRepo, appValidator, log)
	// initialize the handlers
	todoHandler := handlers.NewTodoHandler(todoService, log)
	// Setup HTTP Router

	mainMUx := http.NewServeMux()
	// router := http.NewServeMux()
	// router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	fmt.Fprintln(w, "Tusk is running! (Bootstrapped)")
	// })

	mainMUx.HandleFunc("/todos", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/todos" {
			todoHandler.GetAllTodos(w, r)
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/todos" {
			todoHandler.CreateTodo(w, r)
			return
		}
		web.RespondError(w, appErrors.NotFoundError("Resource not found or method not allowed", nil), http.StatusNotFound)

	})
	mainMUx.HandleFunc("/todos/", func(w http.ResponseWriter, r *http.Request) {
		
			// For /todos/{id} routes, delegate to the specific handler method
			switch r.Method {
			case http.MethodGet:
				todoHandler.GetTodoByID(w, r)
			case http.MethodPut:
				todoHandler.UpdateTodo(w, r)
			case http.MethodDelete:
				todoHandler.DeleteTodo(w, r)
			default:
				web.RespondError(w, appErrors.New("METHOD_NOT_ALLOWED", "Method not allowed for this resource", nil), http.StatusMethodNotAllowed)
			}
			return
		
	})

	//middleware
	var handler http.Handler = mainMUx
	handler = middleware.Recovery(log)(handler)
	handler = middleware.Logger(log)(handler)
	//Start Server
	serverAddr := fmt.Sprintf(":%d", cfg.ServerPort)
	log.Info(fmt.Sprintf("Server starting on %s", serverAddr))
	if err := http.ListenAndServe(serverAddr, handler); err != nil {
		return fmt.Errorf("server failed to start: %w", err)
	}

	return nil

}
