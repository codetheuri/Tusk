package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"math"

	"github.com/codetheuri/todolist/internal/app/services"
	appErrors "github.com/codetheuri/todolist/pkg/errors"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/codetheuri/todolist/pkg/pagination"
	"github.com/codetheuri/todolist/pkg/web"
	"github.com/go-chi/chi"
)

type TodoHandler struct {
	todoService services.TodoService
	log         logger.Logger
}

// instance of the TodoHandler
func NewTodoHandler(svc services.TodoService, log logger.Logger) *TodoHandler {
	return &TodoHandler{
		todoService: svc,
		log:         log,
	}
}

// post todos
func (h *TodoHandler) CreateTodo(w http.ResponseWriter, r *http.Request) {
	h.log.Debug("Handler: Received CreateTodo request")
	var req services.CreateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("handler: Failed to decode request body", "error", err)
		web.RespondError(w, appErrors.New("INVALID_INPUT", "Invalid request body format", err), http.StatusBadRequest)
		return
	}
	//call service
	res, err := h.todoService.CreateTodo(&req)
	if err != nil {
		h.log.Error("Handler: Service call failed", err)
		web.RespondError(w, err, http.StatusInternalServerError)
		return
	}
	web.RespondJSON(w, http.StatusCreated, res)
	h.log.Info("Handler: Todo request handled successfully", "todoID", res.ID)
}

// get todo by id
func (h *TodoHandler) GetTodoByID(w http.ResponseWriter, r *http.Request) {
	h.log.Debug("Hander: Received GetTodoByID request")
	// idStr := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	// id, err := strconv.ParseUint(idStr, 10, 32)
	idStr := chi.URLParam(r, "id")
	// if idStr == "" {
	// 	web.RespondError(w, r, h.Log, errors.NewError(errors.ENonExistent, "ID is missing in the URL"))
	// 	return
	// }

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		h.log.Warn("Handler: Invalid ID format", "id", idStr, "error", err)
		web.RespondError(w, appErrors.ValidationError("Invalid ID format", err, nil), http.StatusBadRequest)
		return
	}
	// Check if the parsed ID is within the bounds of the uint type
	if id > math.MaxUint {
		h.log.Warn("Handler: ID exceeds the maximum allowed value for uint", "id", id)
		web.RespondError(w, appErrors.ValidationError("ID exceeds the maximum allowed value", nil, nil), http.StatusBadRequest)
		return
	}
	res, err := h.todoService.GetTodoByID(uint(id))
	if err != nil {
		h.log.Error("Handler: Service call failed for GetTodoByID", err, "todoID", id)
		web.RespondError(w, err, http.StatusInternalServerError)
		return
	}
	web.RespondJSON(w, http.StatusOK, res)
	h.log.Info("Handler: Todo retrieved successfully", "todoID", res.ID)
}

// get all todos
func (h *TodoHandler) GetAllTodos(w http.ResponseWriter, r *http.Request) {
	h.log.Debug("Handler: Received GetAllTodos request")
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = pagination.DefaultPage
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <1 || limit > pagination.MaxLimit{
		limit = pagination.DefaultLimit
	}

	p, err := h.todoService.GetAllTodos(page,limit)
	if err != nil {
		h.log.Error("Handler: Service call failed for GetAllTodos", err)
		web.RespondError(w, err, http.StatusInternalServerError)
		return
	}
	web.RespondJSON(w, http.StatusOK, p)
	h.log.Info("Handler: Todos retrieved successfully", "page", p.Page, "limit", p.Limit, "total_rows", p.TotalRows)

}

// UpdateTodo
func (h *TodoHandler) UpdateTodo(w http.ResponseWriter, r *http.Request) {
	h.log.Debug("Handler: Received UpdateTodo request")
	// idStr := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.log.Warn("Handler: Invalid ID format in UpdateTodo request", "idStr", idStr, "error", err)
		web.RespondError(w, appErrors.ValidationError("Invalid todo ID format", err, nil), http.StatusBadRequest)
		return
	}

	var req services.UpdateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("Handler: Failed to decode update todo request body", "error", err)
		web.RespondError(w, appErrors.New("INVALID_INPUT", "Invalid request body format", err), http.StatusBadRequest)
		return
	}

	req.ID = uint(id)

	res, err := h.todoService.UpdateTodo(&req)
	if err != nil {
		h.log.Error("Handler: Service call failed for UpdateTodo", err, "todoID", id)
		web.RespondError(w, err, http.StatusInternalServerError)
		return
	}
	web.RespondJSON(w, http.StatusOK, res)
	h.log.Info("Handler: Todo updated successfully", "todoID", res.ID)
}

// DeleteTodo
func (h *TodoHandler) DeleteTodo(w http.ResponseWriter, r *http.Request) {
	h.log.Debug("Handler: received DeleteTodo request")

	// idStr := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.log.Warn("Handler: Invalid ID format in DeleteTodo request", "idStr", idStr, "error", err)
		web.RespondError(w, appErrors.ValidationError("Invalid todo ID format", err, nil), http.StatusBadRequest)
		return
	}
	//call service
	err = h.todoService.DeleteTodo(uint(id))
	if err != nil {
		h.log.Error("Handler: Service call failed for DeleteTodo", err, "todoID", id)
		web.RespondError(w, err, http.StatusInternalServerError)
		return
	}
	web.RespondJSON(w, http.StatusNoContent, nil)
	h.log.Info("Handler: Todo deleted successfully", "todoID", id)
}
