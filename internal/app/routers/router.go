package router

import (
	"net/http"

	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/go-chi/chi"
)

func NewRouter(log logger.Logger) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	log.Info("Base HTTP router initialized. ")
	return r

}

// func NewRouter(h *handlers.TodoHandler, log logger.Logger) http.Handler {
//   r := chi.NewRouter()
//     r.Use(middleware.Logger)  // Log each request
//      r.Use(middleware.RequestID) // Generate a unique ID for each request

// 	r.Use(middleware.Recoverer)

// 	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
//      web.RespondJSON(w, http.StatusOK, map[string]string{"message": "Tusk is running!"})
// 	})
// 	// api versioning
// 	r.Route("/api/v1", func(r chi.Router) {

// 		//within the api version
// 	r.Route("/todos", func(r chi.Router) {
// 		r.Get("/", h.GetAllTodos)
// 		r.Post("/", h.CreateTodo)

// 		r.Get("/{id}", h.GetTodoByID)
// 		r.Put("/{id}", h.UpdateTodo)
// 		r.Get("/all/", h.GetAllIncludingDeleted)
// 			r.Get("/all", h.GetAllIncludingDeleted)
// 		r.Delete("/{id}", h.SoftDeleteTodo)
// 		r.Patch("/{id}", h.RestoreTodo)
// 		r.Delete("/hard/{id}", h.HardDeleteTodo)

// 	})
// 	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
// 		log.Warn("Route not found", "path", r.URL.Path, "method", r.Method)
// 		// web.RespondError(w, "The requested resource was not found", http.StatusNotFound)
// 		web.RespondJSON(w, http.StatusNotFound, map[string]string{"error": "Resource not found"})
// 		// http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
// 	})
// 	})

// 	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
// 		log.Warn("Method not allowed", "path", r.URL.Path, "method", r.Method)
// 		web.RespondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
// 		// http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
// 	})

// 	return r

// }

// import (
// 	"encoding/json"
// 	"net/http"
// 	"strconv"
// 	"strings"

// 	handler "github.com/codetheuri/todolist/internal/app/handlers"
// 	"github.com/gorilla/mux"
// )

// func SetupRouter() {

// 	r := mux.NewRouter()
// 	r.HandleFunc("/todos", handler.GetTodos).Methods("GET")
// 	r.HandleFunc("/todos", handler.AddTodo).Methods("POST")
// 	// r.HandleFunc("/todos/{id}", handler.GetOneTodo).Methods("GET")
// 	// r.HandleFunc("/todos/{id:[0-9]+}", handler.UpdateTodo).Methods("PUT")
// 	// r.HandleFunc("/todos/{id:[0-9]+}", handler.DeleteTodo).Methods("DELETE")

// 	// return r
// 	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
// 		switch r.Method {
// 		case http.MethodGet:
// 			handler.GetTodos(w, r)
// 		case http.MethodPost:
// 			handler.AddTodo(w, r)
// 		default:
// 			w.WriteHeader(http.StatusMethodNotAllowed)
// 			json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
// 		}

// 	})
// 	http.HandleFunc("/todos/", func(w http.ResponseWriter, r *http.Request) {
// 		idStr := strings.TrimPrefix(r.URL.Path, "/todos/")
// 		id, err := strconv.Atoi(idStr)
// 		if err != nil {
// 			http.Error(w, "Invalid ID", http.StatusBadRequest)
// 			return
// 		}
// 		switch r.Method {
// 		case http.MethodGet:
// 			handler.GetOneTodo(w, r, id)
// 		case http.MethodPut:
// 			handler.UpdateTodo(w, r, id)
// 		case http.MethodDelete:
// 			handler.DeleteTodo(w, r, id)
// 		default:
// 			w.WriteHeader(http.StatusMethodNotAllowed)
// 			json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})

// 		}

// 	})
// 	http.HandleFunc("/todos", func(w http.ResponseWriter, r *http.Request) {

// 		if r.Method == http.MethodPost {
// 			handler.AddTodo(w, r)
// 		} else {
// 			w.WriteHeader(http.StatusMethodNotAllowed)
// 			json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
// 		}
// 	})
// }
