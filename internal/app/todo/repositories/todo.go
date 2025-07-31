package repositories

import (
	"errors"
	"fmt"

	"github.com/codetheuri/todolist/internal/app/todo/models"
	appErrors "github.com/codetheuri/todolist/pkg/errors"
	"github.com/codetheuri/todolist/pkg/logger"
	"golang.org/x/net/context"
	"gorm.io/gorm"
)

// define the TodoRepository interface
type TodoRepository interface {
	CreateTodo(ctx context.Context, todo *models.Todo) (*models.Todo, error)
	GetTodoByID(id uint) (*models.Todo, error)
	GetAllTodos(ctx context.Context, offset, limit int) ([]models.Todo, int64, error)
	UpdateTodo(todo *models.Todo) (*models.Todo, error)
	GetAllIncludingDeleted(ctx context.Context, offset, limit int) ([]models.Todo, int64, error)
	SoftDeleteTodo(id uint) error
	RestoreTodo(id uint) error
	HardDeleteTodo(id uint) error
}

// implement the TodoRepository interface
type gormTodoRepository struct {
	db  *gorm.DB
	log logger.Logger
}

// NewGormTodoRepository creates a new instance of gormTodoRepository
func NewGormTodoRepository(db *gorm.DB, log logger.Logger) TodoRepository {
	return &gormTodoRepository{
		db:  db,
		log: log,
	}
}

// create a new todo
func (r *gormTodoRepository) CreateTodo(ctx context.Context, todo *models.Todo) (*models.Todo, error) {
	if err := r.db.WithContext(ctx).Create(todo).Error; err != nil {
		r.log.Error("failed to create todo", err, "todo", todo)
		return nil, appErrors.DatabaseError("failed to create todo", err)
	}
	return todo, nil
}

// retrieve a todo by ID
func (r *gormTodoRepository) GetTodoByID(id uint) (*models.Todo, error) {
	var todo models.Todo
	if err := r.db.First(&todo, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warn("todo not found", "id", id)
			return nil, appErrors.NotFoundError(fmt.Sprintf("todo with id %d not found", id), err)
		}
		r.log.Error("failed to get todo by id", err, "id", id)
		return nil, appErrors.DatabaseError(fmt.Sprintf("failed to get todo by id %d", id), err)
	}
	r.log.Debug("todo retrieved successfully", "id", id)
	return &todo, nil
}

// retrieve all todos
func (r *gormTodoRepository) GetAllTodos(ctx context.Context, offset, limit int) ([]models.Todo, int64, error) {
	var todos []models.Todo
	var totalCount int64
	//count records
	if err := r.db.WithContext(ctx).Model(&models.Todo{}).Count(&totalCount).Error; err != nil {
		r.log.Error("Repository: Failed to count todos", err)
		return nil, 0, err
	}
	//fetch todos with pagination
	if err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&todos).Error; err != nil {
		r.log.Error("Repository: Failed to fetch all todos", err)
		return nil, 0, err
	}

	return todos, totalCount, nil
}

// update a todo by ID
func (r *gormTodoRepository) UpdateTodo(todo *models.Todo) (*models.Todo, error) {
	existingTodo := &models.Todo{}
	if err := r.db.First(existingTodo, todo.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warn("todo not found for update", "id", todo.ID)
			return nil, appErrors.NotFoundError(fmt.Sprintf("todo with id %d not found", todo.ID), err)
		}
		r.log.Error("failed to find todo for update", err, "id", todo.ID)
		return nil, appErrors.DatabaseError(fmt.Sprintf("failed to find todo with id %d for update", todo.ID), err)
	}
	existingTodo.Title = todo.Title
	existingTodo.Description = todo.Description
	existingTodo.Completed = todo.Completed

	if err := r.db.Save(existingTodo).Error; err != nil {
		r.log.Error("failed to update todo", err, "todo", todo)
		return nil, appErrors.DatabaseError("failed to update todo", err)
	}
	r.log.Info("todo updated successfully", "id", existingTodo.ID)
	return todo, nil
}
func (r *gormTodoRepository) GetAllIncludingDeleted(ctx context.Context, offset, limit int) ([]models.Todo, int64, error) {
	var todos []models.Todo
	var totalCount int64
	if err := r.db.Unscoped().WithContext(ctx).Model(&models.Todo{}).Count(&totalCount).Error; err != nil {
		r.log.Error("Repository: Failed to count todos", err)
		return nil, 0, err
	}
	//fetch todos with pagination
	if err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&todos).Error; err != nil {
		r.log.Error("Repository: Failed to fetch all todos", err)
		return nil, 0, err
	}

	return todos, totalCount, nil
}

// delete a todo by ID
func (r *gormTodoRepository) SoftDeleteTodo(id uint) error {
	_, err := r.GetTodoByID(id)
	if err != nil {
		return err // if todo not found, return the error
	}
	if err := r.db.Delete(&models.Todo{}, id).Error; err != nil {
		r.log.Error("failed to delete todo", err, "id", id)
		return appErrors.DatabaseError("failed to delete todo", err)
	}
	r.log.Info("todo deleted successfully", "id", id)
	return nil
}

func (r *gormTodoRepository) RestoreTodo(id uint) error {
	var todo models.Todo
	if err := r.db.Unscoped().First(&todo, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warn("todo not found for restore", "id", id)
			return appErrors.NotFoundError(fmt.Sprintf("todo with id %d not found", id), err)
		}
		r.log.Error("failed to find todo for restore", err, "id", id)
		return appErrors.DatabaseError(fmt.Sprintf("failed to find todo with id %d for restore", id), err)
	}
	todo.DeletedAt = gorm.DeletedAt{} // reset the DeletedAt field to restore the record
	// if err := r.db.Model(&todo).Update("deleted_at", nil).Error; err != nil {
	// 	r.log.Error("failed to restore todo", err, "id", id)
	// 	return appErrors.DatabaseError("failed to restore todo", err)
	// }
	if err := r.db.Save(&todo).Error; err != nil {
		r.log.Error("failed to restore todo", err, "id", id)
		return appErrors.DatabaseError("failed to restore todo", err)
	}
	r.log.Info("todo restored successfully", "id", id)
	return nil
}
func (r *gormTodoRepository) HardDeleteTodo(id uint) error {
	var todo models.Todo
	if err := r.db.Unscoped().First(&todo, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warn("todo not found for hard delete", "id", id)
			return appErrors.NotFoundError(fmt.Sprintf("todo with id %d not found", id), err)
		}
		r.log.Error("failed to find todo for hard delete", err, "id", id)
		return appErrors.DatabaseError(fmt.Sprintf("failed to find todo with id %d for hard delete", id), err)
	}

	if err := r.db.Unscoped().Delete(&todo).Error; err != nil {
		r.log.Error("failed to hard delete todo", err, "id", id)
		return appErrors.DatabaseError("failed to hard delete todo", err)
	}
	r.log.Info("todo hard deleted successfully", "id", id)
	return nil
}
