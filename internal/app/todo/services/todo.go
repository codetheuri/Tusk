package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/codetheuri/todolist/internal/app/todo/models"
	"github.com/codetheuri/todolist/internal/app/todo/repositories"
	appErrors "github.com/codetheuri/todolist/pkg/errors"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/codetheuri/todolist/pkg/pagination"
	"github.com/codetheuri/todolist/pkg/validators"
)

// interface
type TodoService interface {
	CreateTodo(ctx context.Context,createReq *CreateTodoRequest) (*TodoResponse, error)
	GetTodoByID(id uint) (*TodoResponse, error)
	GetAllTodos(page, limit int) (*pagination.Pagination, error)
	UpdateTodo(updateReq *UpdateTodoRequest) (*TodoResponse, error)
	GetAllIncludingDeleted(page, limit int) (*pagination.Pagination, error)
	SoftDeleteTodo(id uint) error
	RestoreTodo(id uint) error
	HardDeleteTodo(id uint) error
}

// implement dtos
type CreateTodoRequest struct {
	Title       string `json:"title" validate:"required,min=3,max=100"`
	Description string `json:"description" validate:"required,max=255"`
	Completed   bool   `json:"completed"`
}
type UpdateTodoRequest struct {
	ID          uint   `json:"id" validate:"required"`
	Title       string `json:"title" validate:"required,min=3,max=100"`
	Description string `json:"description" validate:"required,max=255"`
	Completed   bool   `json:"completed"`
}

type TodoResponse struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
	DeletedAt   string `json:"deleted_at,omitempty"` 
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// implement TodoService interface
type todoService struct {
	repo      repositories.TodoRepository
	validator *validators.Validator
	log       logger.Logger
}

// new todo service instance
func NewTodoService(repo repositories.TodoRepository, validator *validators.Validator, log logger.Logger) TodoService {
	return &todoService{
		repo:      repo,
		validator: validator,
		log:       log,
	}
}

// CreateTodo
func (s *todoService) CreateTodo(ctx context.Context,createReq *CreateTodoRequest) (*TodoResponse, error) {
	//validate
	fieldErrors := s.validator.Struct(createReq)
	if fieldErrors != nil {
		s.log.Warn("validation failed for create todo request", "error", fieldErrors)
		return nil, appErrors.ValidationError("invalid todo data", nil, fieldErrors)
	}

	//logic

	todo := &models.Todo{
		Title:       createReq.Title,
		Description: createReq.Description,
		Completed:   createReq.Completed,
	}
	//persist
	createdTodo, err := s.repo.CreateTodo(ctx,todo)
	if err != nil {
		s.log.Error("service: failed to create todo in repository", err)

		var dbErr appErrors.AppError
		if errors.As(err, &dbErr) && dbErr.Code() == "DATABASE_ERROR" {
			return nil, appErrors.New("CREATE_FAILED", "failed to create todo due to database issue", err)
		}
		return nil, err
	}
	return s.toTodoResponse(createdTodo), nil
}
func (s *todoService) GetTodoByID(id uint) (*TodoResponse, error) {
	//fetch

	todo, err := s.repo.GetTodoByID(id)
	if err != nil {
		s.log.Error("service: failed to get todo by id", err, "id", id)
		var notFoundErr appErrors.AppError
		if errors.As(err, &notFoundErr) && notFoundErr.Code() == "NOT_FOUND" {
			return nil, appErrors.NotFoundError(fmt.Sprintf("todo with id %d not found", id), err)
		}
		return nil, err
	}
	//map to response
	return s.toTodoResponse(todo), nil
}

// get all
func (s *todoService) GetAllTodos(page, limit int) (*pagination.Pagination, error) {
	//fetch
	p := &pagination.Pagination{Page: page, Limit: limit}
	todos, err := s.repo.GetAllTodos(p)

	if err != nil {
		s.log.Error("service : failed to get all todos ", err)
		return nil, err
	}
	//map models
	todoResponses := make([]TodoResponse, len(todos))
	for i, todo := range todos {
		todoResponses[i] = TodoResponse{
			ID:          todo.ID,
			Title:       todo.Title,
			Description: todo.Description,
			Completed:   todo.Completed,
			DeletedAt:  todo.DeletedAt.Time.Format("2006-01-02 15:04:05"),
			CreatedAt:   todo.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:   todo.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
	}
	rowsInterface := make([]interface{}, len(todoResponses))
	for i, v := range todoResponses {
		rowsInterface[i] = v
	}
	p.Rows = rowsInterface
	return p, nil
}

// update
func (s *todoService) UpdateTodo(updateReq *UpdateTodoRequest) (*TodoResponse, error) {
	//validate\
	fieldErrors := s.validator.Struct(updateReq)
	if fieldErrors != nil {
		s.log.Warn("validation failed ", "error", fieldErrors)
		return nil, appErrors.ValidationError("invalid todo data", nil, fieldErrors)
	}

	//fetch existing
	existingTodo, err := s.repo.GetTodoByID(updateReq.ID)
	if err != nil {
		s.log.Error("service : failed to get data", err, "id", updateReq.ID)
		var notFoundErr appErrors.AppError
		if errors.As(err, &notFoundErr) && notFoundErr.Code() == "NOT_FOUND" {
			return nil, appErrors.NotFoundError(fmt.Sprintf("todo with id %d not found", updateReq.ID), err)
		}
		return nil, err
	}
	//update fields
	if updateReq.Title != "" {
		existingTodo.Title = updateReq.Title
	}
	if updateReq.Description != "" {
		existingTodo.Description = updateReq.Description
	}
	// if updateReq.Completed is false, we don't update it
	if updateReq.Completed != existingTodo.Completed {
		existingTodo.Completed = updateReq.Completed
	}
	//persist
	updatedTodo, err := s.repo.UpdateTodo(existingTodo)
	if err != nil {
		s.log.Error("service: failed to update todo in repository", err, "id", existingTodo.ID)
		var dbErr appErrors.AppError
		if errors.As(err, &dbErr) && dbErr.Code() == "DATABASE_ERROR" {
			return nil, appErrors.New("UPDATE_FAILED", "failed to update due to database issue", err)
		}
		return nil, err
	}
	return s.toTodoResponse(updatedTodo), nil

}
// get all including deleted
func (s *todoService) GetAllIncludingDeleted(page, limit int) (*pagination.Pagination, error) {

	p := &pagination.Pagination{Page: page, Limit: limit}
	todos, err := s.repo.GetAllIncludingDeleted(p)
	if err != nil {
		s.log.Error("service: failed to get all todos including deleted", err)
		return nil, err
	}
	
	todoResponses := make([]TodoResponse, len(todos))
	for i, todo := range todos {
		todoResponses[i] = TodoResponse{
			ID:          todo.ID,
			Title:       todo.Title,
			Description: todo.Description,
			Completed:   todo.Completed,
			DeletedAt:   todo.DeletedAt.Time.Format("2006-01-02 15:04:05"),
			CreatedAt:   todo.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:   todo.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
	}	
	rowsInterface := make([]interface{}, len(todoResponses))
	for i, v := range todoResponses {
		rowsInterface[i] = v
	}
	p.Rows = rowsInterface
	return p, nil
}

// soft delete
func (s *todoService) SoftDeleteTodo(id uint) error {
	// call
	err := s.repo.SoftDeleteTodo(id)
	if err != nil {
		s.log.Error("serrvice: failed to delete todo from repository", err, "id", id)
		var notFoundErr appErrors.AppError
		if errors.As(err, &notFoundErr) && notFoundErr.Code() == "NOT_FOUND" {
			return appErrors.NotFoundError(fmt.Sprintf("todo with  ID %d not found", id), err)
		}
		return err
	}
	return nil
}
func (s *todoService) RestoreTodo(id uint) error {

	err := s.repo.RestoreTodo(id)
	if err != nil {
		s.log.Error("service: failed to restore todo from repository", err, "id", id)
		var notFoundErr appErrors.AppError
		if errors.As(err, &notFoundErr) && notFoundErr.Code() == "NOT_FOUND" {
			return appErrors.NotFoundError(fmt.Sprintf("todo with ID %d not found", id), err)
		}
		return err
	}
	return nil
}

func (s *todoService) HardDeleteTodo(id uint) error {
	err := s.repo.HardDeleteTodo(id)
	if err != nil {
		s.log.Error("service: failed to hard delete todo from repository", err, "id", id)
		var notFoundErr appErrors.AppError
		if errors.As(err, &notFoundErr) && notFoundErr.Code() == "NOT_FOUND" {
			return appErrors.NotFoundError(fmt.Sprintf("todo with ID %d not found", id), err)
		}
		return err
	}
	return nil
}
// helper convert models.Todo to TodoResponse
func (s *todoService) toTodoResponse(todo *models.Todo) *TodoResponse {
	return &TodoResponse{
		ID:          todo.ID,
		Title:       todo.Title,
		Description: todo.Description,
		Completed:   todo.Completed,
		DeletedAt:  todo.DeletedAt.Time.Format("2006-01-02 15:04:05"),
		CreatedAt:   todo.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   todo.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
