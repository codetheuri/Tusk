package bootstrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/codetheuri/todolist/config"
	"github.com/codetheuri/todolist/internal/app/models"
	"github.com/codetheuri/todolist/internal/app/repositories"
	"github.com/codetheuri/todolist/internal/platform/database"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/codetheuri/todolist/pkg/web"
)

// initiliazes and start the application
func Run(cfg *config.Config, log logger.Logger) error {
	//db
	db, err := database.NewGoRMDB(cfg, log)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Info("Running migrations...")
	
	if err := db.AutoMigrate(&models.Todo{}); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	log.Info("Migrations completed successfully")

	//initialize the router

	//initilialize app components
	// appValidator := validator.NewValidator()

	//initialize the repositories
	todoRepo := repositories.NewGormTodoRepository(db, log)
	router := http.NewServeMux()
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Tusk is running! (Bootstrapped)")
	})

	router.HandleFunc("/test-repo", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
case http.MethodPost:
			newTodo := &models.Todo{Title: "Test Todo", Description: "Created from /test-repo"}
			createdTodo, err := todoRepo.CreateTodo(newTodo)
			if err != nil {
				web.RespondError(w, errors.New("Failed to create test todo"), http.StatusInternalServerError)
				return
			}
			web.RespondJSON(w, http.StatusCreated, createdTodo)
		case http.MethodGet:
			todos, err := todoRepo.GetAllTodos()
			if err != nil {
				web.RespondError(w, errors.New("Failed to get test todos"), http.StatusInternalServerError)
				return
			}
			web.RespondJSON(w, http.StatusOK, todos)
		case http.MethodPut:
			var updatedTodo models.Todo
			if err := json.NewDecoder(r.Body).Decode(&updatedTodo); err != nil {
				web.RespondError(w, errors.New("Invalid request payload"), http.StatusBadRequest)
				return
			}
			if updatedTodo.ID == 0 {
				web.RespondError(w, errors.New("ID is required for update"), http.StatusBadRequest)
				return
			}
			resultTodo, err := todoRepo.UpdateTodo(&updatedTodo)
			if err != nil {
				web.RespondError(w, err, http.StatusInternalServerError)
				return
			}
			web.RespondJSON(w, http.StatusOK, resultTodo)
		case http.MethodDelete:
			idStr := r.URL.Query().Get("id")
			if idStr == "" {
				web.RespondError(w, errors.New("ID is required for deletion"), http.StatusBadRequest)
				return
			}
			id, err := strconv.ParseUint(idStr, 10, 32)
			if err != nil {
				web.RespondError(w, errors.New("Invalid ID format"), http.StatusBadRequest)
				return
			}
			if err := todoRepo.DeleteTodo(uint(id)); err != nil {
				web.RespondError(w, err, http.StatusInternalServerError)
				return
			}
			web.RespondJSON(w, http.StatusNoContent, nil)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		}
	})

	// 4. Start Server
	serverAddr := fmt.Sprintf(":%d", cfg.ServerPort)
	log.Info(fmt.Sprintf("Server starting on %s", serverAddr))
	if err := http.ListenAndServe(serverAddr, router); err != nil {
		return fmt.Errorf("server failed to start: %w", err)
	}

	return nil

}
