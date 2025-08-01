package handlers

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/codetheuri/todolist/internal/app/todo/services"
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
	ctx, cancel := context.WithTimeout(r.Context(), 5* time.Second)
	defer cancel()
	res, err := h.todoService.CreateTodo(ctx,&req)
	if err != nil {
		h.log.Error("Handler: Service call failed", err)
		web.RespondError(w, err, http.StatusInternalServerError)
		return
	}
	
	
	web.RespondData(w,http.StatusCreated,res, "todo created succssfully")
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
	
	web.RespondData(w,http.StatusOK, res, "")
	h.log.Info("Handler: Todo retrieved successfully", "todoID", res.ID)
}

// get all todos
func (h *TodoHandler) GetAllTodos(w http.ResponseWriter, r *http.Request) {
	h.log.Debug("Handler: Received GetAllTodos request")
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		page = pagination.DefaultPage
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil{
		limit = pagination.DefaultLimit
	}
    ctx := r.Context()
	p, err := h.todoService.GetAllTodos(ctx,page,limit)
	if err != nil {
		h.log.Error("Handler: Service call failed for GetAllTodos", err)
		web.RespondError(w, err, http.StatusInternalServerError)
		return
	}
	// web.RespondJSON(w, http.StatusOK, p)
	web.RespondListData(w,http.StatusOK,p.Data,p.Metadata)
	h.log.Info("Handler: Todos retrieved successfully", "page", p.Metadata.Page, "limit", p.Metadata.Limit, "total_items", p.Metadata.TotalItems)

}

func (h *TodoHandler) GetAllIncludingDeleted(w http.ResponseWriter, r *http.Request) {
	h.log.Debug("Handler: Received GetAllIncludingDeleted request")
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		page = pagination.DefaultPage
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil{
		limit = pagination.DefaultLimit
	}
    ctx := r.Context()
	p, err := h.todoService.GetAllIncludingDeleted(ctx,page, limit)
	if err != nil {
		h.log.Error("Handler: Service call failed for GetAllIncludingDeleted", err)
		web.RespondError(w, err, http.StatusInternalServerError)
		return
	}
	// web.RespondJSON(w, http.StatusOK, p)
	web.RespondListData(w,http.StatusOK,p.Data,p.Metadata)
	// h.log.Info("Handler: Todos including deleted retrieved successfully", "page", p.Metadata.Page, "limit", p.Metadata.Limit, "total_item", p.Metadata.TotalItems)
}
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
	
	web.RespondData(w,http.StatusOK, res, "Todo updated successfully")
	h.log.Info("Handler: Todo updated successfully", "todoID", res.ID)
}

// DeleteTodo
func (h *TodoHandler) SoftDeleteTodo(w http.ResponseWriter, r *http.Request) {
	h.log.Debug("Handler: received DeleteTodo request")


	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.log.Warn("Handler: Invalid ID format in DeleteTodo request", "idStr", idStr, "error", err)
		web.RespondError(w, appErrors.ValidationError("Invalid todo ID format", err, nil), http.StatusBadRequest)
		return
	}
	//call service
	err = h.todoService.SoftDeleteTodo(uint(id))
	if err != nil {
		h.log.Error("Handler: Service call failed for DeleteTodo", err, "todoID", id)
		web.RespondError(w, err, http.StatusInternalServerError)
		return
	}
	
	
	 web.RespondMessage(w, http.StatusOK, "Todo soft-deleted successfully", "success", "toast")
	h.log.Info("Handler: Todo deleted successfully", "todoID", id)
}
func (h *TodoHandler) RestoreTodo(w http.ResponseWriter, r *http.Request) {
	h.log.Debug("Handler: Received RestoreTodo request")
	// idStr := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.log.Warn("Handler: Invalid ID format in RestoreTodo request", "idStr", idStr, "error", err)
		web.RespondError(w, appErrors.ValidationError("Invalid todo ID format", err, nil), http.StatusBadRequest)
		return
	}
	//call service
	err = h.todoService.RestoreTodo(uint(id))
	if err != nil {
		h.log.Error("Handler: Service call failed for RestoreTodo", err, "todoID", id)
		web.RespondError(w, err, http.StatusInternalServerError)
		return
	}
	
	web.RespondMessage(w,http.StatusOK, "Todo restored successfully", "success", "alert")
	h.log.Info("Handler: Todo restored successfully", "todoID", id)
}
func (h *TodoHandler) HardDeleteTodo(w http.ResponseWriter, r *http.Request) {
	h.log.Debug("Handler: Received HardDeleteTodo request")
	// idStr := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.log.Warn("Handler: Invalid ID format in HardDeleteTodo request", "idStr", idStr, "error", err)
		web.RespondError(w, appErrors.ValidationError("Invalid todo ID format", err, nil), http.StatusBadRequest)
		return
	}

	err = h.todoService.HardDeleteTodo(uint(id))
	if err != nil {
		h.log.Error("Handler: Service call failed for HardDeleteTodo", err, "todoID", id)
		web.RespondError(w, err, http.StatusInternalServerError)
		return
	}
	
	  w.WriteHeader(http.StatusNoContent)
	h.log.Info("Handler: Todo hard deleted successfully", "todoID", id)
}
