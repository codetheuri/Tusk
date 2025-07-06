package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/codetheuri/todolist/config"
	"github.com/codetheuri/todolist/internal/app/models"
	"github.com/codetheuri/todolist/internal/app/repositories"
	"github.com/codetheuri/todolist/internal/app/services"
	"github.com/codetheuri/todolist/internal/platform/database"
	appErrors "github.com/codetheuri/todolist/pkg/errors"
	"github.com/codetheuri/todolist/pkg/logger"

	// "github.com/codetheuri/todolist/pkg/validators"
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
	// appValidator := validator.NewValidator()

	//initialize the repositories
	todoRepo := repositories.NewGormTodoRepository(db, log)
	//initilliaze services
	todoService := services.NewTodoService(todoRepo,  log)

	router := http.NewServeMux()
	// router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	fmt.Fprintln(w, "Tusk is running! (Bootstrapped)")
	// })

	router.HandleFunc("/test-service", func(w http.ResponseWriter, r *http.Request) {
		// Log the request method for debugging
		log.Debug("Received request to /test-service", "method", r.Method)

		switch r.Method {
		case http.MethodPost:
			var req services.CreateTodoRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				web.RespondError(w, appErrors.New("INVALID_INPUT", "Invalid request body", err), http.StatusBadRequest)
				return
			}
			createdTodo, err := todoService.CreateTodo(&req) // <--- Call Service method!
			if err != nil {
				log.Error("Failed to create todo via service", err)
				web.RespondError(w, err, http.StatusInternalServerError)
				return
			}
			web.RespondJSON(w, http.StatusCreated, createdTodo)

		case http.MethodGet:
			// Check for ID in query parameter for GetById, otherwise GetAll
			idStr := r.URL.Query().Get("id")
			if idStr != "" {
				id, err := strconv.ParseUint(idStr, 10, 32)
				if err != nil {
					web.RespondError(w, appErrors.New("INVALID_INPUT", "Invalid ID format", err), http.StatusBadRequest)
					return
				}
				todo, err := todoService.GetTodoByID(uint(id)) // <--- Call Service method!
				if err != nil {
					web.RespondError(w, err, http.StatusInternalServerError)
					return
				}
				web.RespondJSON(w, http.StatusOK, todo)
			} else {
				todos, err := todoService.GetAllTodos() // <--- Call Service method!
				if err != nil {
					web.RespondError(w, err, http.StatusInternalServerError)
					return
				}
				web.RespondJSON(w, http.StatusOK, todos)
			}

		case http.MethodPut:
			var req services.UpdateTodoRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				web.RespondError(w, appErrors.New("INVALID_INPUT", "Invalid request body", err), http.StatusBadRequest)
				return
			}
			updatedTodo, err := todoService.UpdateTodo(&req) // <--- Call Service method!
			if err != nil {
				web.RespondError(w, err, http.StatusInternalServerError)
				return
			}
			web.RespondJSON(w, http.StatusOK, updatedTodo)

		case http.MethodDelete:
			idStr := r.URL.Query().Get("id")
			if idStr == "" {
				web.RespondError(w, appErrors.New("INVALID_INPUT", "ID query parameter is required for delete", nil), http.StatusBadRequest)
				return
			}
			id, err := strconv.ParseUint(idStr, 10, 32)
			if err != nil {
				web.RespondError(w, appErrors.New("INVALID_INPUT", "Invalid ID format", err), http.StatusBadRequest)
				return
			}
			if err := todoService.DeleteTodo(uint(id)); err != nil { // <--- Call Service method!
				web.RespondError(w, err, http.StatusInternalServerError)
				return
			}
			web.RespondJSON(w, http.StatusNoContent, nil) // 204 No Content

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
