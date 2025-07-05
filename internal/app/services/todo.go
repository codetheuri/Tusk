package services

import (
	"github.com/codetheuri/todolist/internal/app/repositories"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/go-playground/validator/v10"
)

// interface
type TodoService interface {
	CreateTodo(createReq *CreateTodoRequest) (*TodoResponse, error)
	GetTodoByID(id uint) (*TodoResponse, error)
	GetAllTodos() ([]TodoResponse, error)
	UpdateTodo(updateReq *UpdateTodoRequest) (*TodoResponse, error)
	DeleteTodo(id uint) error
}

// implement dtos
type CreateTodoRequest struct {
	Title       string `json:"title" validate:"required,min=3,max=100"`
	Description string `json:"description" validate:"required,max=255"`
	Completed   bool   `json:"completed"`
}
type UpdateTodoRequest struct {
	ID          uint   `json:"id" validate:"required"`
	Title       string `json:"title" validate:"omniempty,min=3,max=100"`
	Description string `json:"description" validate:"omniempty,required,max=255"`
	Completed   bool   `json:"completed"`
}

type TodoResponse struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}
type todoService struct {
	repo      repositories.TodoRepository
	validator validator.Validate
	log       logger.Logger
}

