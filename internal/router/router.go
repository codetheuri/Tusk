package router

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/codetheuri/todolist/internal/handler"
)

func SetupRouter() {

	// r := mux.NewRouter()
	// r.HandleFunc("/", handler.GetTodos).Methods("GET")
	// r.HandleFunc("/todos", handler.AddTodo).Methods("POST")
	// r.HandleFunc("/todos/{id}", handler.GetOneTodo).Methods("GET")
	// r.HandleFunc("/todos/{id:[0-9]+}", handler.UpdateTodo).Methods("PUT")
	// r.HandleFunc("/todos/{id:[0-9]+}", handler.DeleteTodo).Methods("DELETE")

	// return r
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	     switch r.Method {
		 case http.MethodGet:
			handler.GetTodos(w,r)
		 case http.MethodPost:
			handler.AddTodo(w,r)
			default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})		
		 }
		
	})
	http.HandleFunc("/todos/",func(w http.ResponseWriter,r *http.Request){
		idStr := strings.TrimPrefix(r.URL.Path, "/todos/")
		id, err := strconv.Atoi(idStr)
		   if err != nil {
            http.Error(w, "Invalid ID", http.StatusBadRequest)
            return
        }
		switch r.Method{
		case http.MethodGet:
			handler.GetOneTodo(w,r,id)
		case http.MethodPut:
			handler.UpdateTodo(w,r,id)
	    case http.MethodDelete:
			handler.DeleteTodo(w,r,id)	
	    default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
					
		}
		
	}) 
	http.HandleFunc("/todos", func(w http.ResponseWriter, r *http.Request) {
	
		if r.Method == http.MethodPost {
			handler.AddTodo(w, r)
		} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		}
	})
}
