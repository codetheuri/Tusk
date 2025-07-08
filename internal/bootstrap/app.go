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
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/codetheuri/todolist/pkg/middleware"
	"github.com/codetheuri/todolist/pkg/validators"
	"github.com/codetheuri/todolist/internal/app/routers"
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
	mainRouter := router.NewRouter(todoHandler, log)
	


	//middleware
	var handler http.Handler = mainRouter
	handler = middleware.Logger(log)(handler)
	handler = middleware.Recovery(log)(handler)
	handler = middleware.RequestID()(handler)

	//Start Server
	serverAddr := fmt.Sprintf(":%d", cfg.ServerPort)
	log.Info(fmt.Sprintf("Server starting on %s", serverAddr))
	if err := http.ListenAndServe(serverAddr, handler); err != nil {
		return fmt.Errorf("server failed to start: %w", err)
	}

	return nil

}
