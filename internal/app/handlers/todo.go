package handler

import (
	"encoding/json"
	"net/http"

	"github.com/codetheuri/todolist/pkg/validators"
)

// var validate = validator.New()

func GetTodos(w http.ResponseWriter, r *http.Request) {
	todos, err := repositories.GetAllTodos()
	if err != nil {
		http.Error(w, "something went wrong", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todos)
}

func AddTodo(w http.ResponseWriter, r *http.Request) {

	var todo model.Todo
	if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request payload"})

		return
	}
	if !validators.ValidateAndRespond(w, &todo) {
		return
	}
	// if err := validate.Struct(todo); err != nil {
	// 	w.WriteHeader(http.StatusUnprocessableEntity)
	// 	json.NewEncoder(w).Encode(map[string]string{"error": "Validation failed", "message": err.Error()})
	// 	return
	// }
	// if todo.Title == "" {
	// 	w.WriteHeader(http.StatusBadRequest)
	// 	json.NewEncoder(w).Encode(map[string]string{"error": "Title is required"})
	// 	return
	// }
	// No need to check for nil since Completed is a bool, not a pointer
	// If you want to ensure a default value, you can leave this out because the zero value for bool is false
	id, err := repository.AddTodo(todo)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to add todo"})

		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int64{"id": id})
	json.NewEncoder(w).Encode(map[string]string{"message": "Task added successfully"})
}
func GetOneTodo(w http.ResponseWriter, r *http.Request, id int) {
	// vars := mux.Vars(r)
	// idStr := vars["id"]
	// id, err := strconv.Atoi(idStr)
	// if err != nil {
	// 	http.Error(w, "Invalid ID", http.StatusBadRequest)
	// 	return
	// }
	todo, err := repository.GetOneTodo(id)
	if err != nil {
		http.Error(w, "Todo not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todo)
}

func UpdateTodo(w http.ResponseWriter, r *http.Request, id int) {
	// vars := mux.Vars(r)
	// idStr := vars["id"]
	// id, err := strconv.Atoi(idStr)
	// if err != nil {
	// 	http.Error(w, "Invalid todo ID", http.StatusBadRequest)
	// 	return
	// }
	var todo model.Todo
	if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	if !validators.ValidateAndRespond(w, &todo) {
		return
	}
	if err := repository.UpdateTodo(id, todo); err != nil {
		http.Error(w, "Failed to update todo", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(todo)
	json.NewEncoder(w).Encode(map[string]string{"message": "Todo updated successfully"})

}

func DeleteTodo(w http.ResponseWriter, r *http.Request, id int) {
	// vars := mux.Vars(r)
	// idStr := vars["id"]
	// id, err := strconv.Atoi(idStr)
	// if err != nil {
	// 	http.Error(w, "Invalid todo ID", http.StatusBadRequest)
	// 	return
	// }

	if err := repository.DeleteTodo(id); err != nil {
		http.Error(w, "Failed to delete todo", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	w.Write([]byte("Todo deleted successfully"))
}
