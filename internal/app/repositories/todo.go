package repositories

import (
	"errors"
	"fmt"

	"github.com/codetheuri/todolist/internal/app/models"
	appErrors "github.com/codetheuri/todolist/pkg/errors"
	"github.com/codetheuri/todolist/pkg/logger"
	"github.com/codetheuri/todolist/pkg/pagination"
	"gorm.io/gorm"
)

// define the TodoRepository interface
type TodoRepository interface {
	CreateTodo(todo *models.Todo) (*models.Todo, error)
	GetTodoByID(id uint) (*models.Todo, error)
	GetAllTodos(p *pagination.Pagination) ([]models.Todo, error)
	UpdateTodo(todo *models.Todo) (*models.Todo, error)
	DeleteTodo(id uint) error
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
func (r *gormTodoRepository) CreateTodo(todo *models.Todo) (*models.Todo, error) {
	if err := r.db.Create(todo).Error; err != nil {
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
	r.log.Debug("todo retrieved successfully", "id", id, )
	return &todo, nil
}

// retrieve all todos
func (r *gormTodoRepository) GetAllTodos(p *pagination.Pagination) ([]models.Todo, error) {
	var todos []models.Todo
	//  apply pagination
	// result := r.db.Scopes(pagination.Paginate(&models.Todo{},p,r.db)).Find(&todos)

	result := r.db.Model(&models.Todo{}).Scopes(pagination.Paginate(p)).Find(&todos)
	if result.Error != nil{
		r.log.Error("Repository: Failed to fetch all todos", result.Error)
		return nil, appErrors.DatabaseError("Failed to fetch todos", result.Error)
	}

	r.log.Info("todos retrieved successfully", "count", len(todos))
	return todos, nil
}

// update a todo by ID
func (r *gormTodoRepository) UpdateTodo( todo *models.Todo) (*models.Todo, error) {
	existingTodo := &models.Todo{}
    if err := r.db.First(existingTodo, todo.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound){
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

// delete a todo by ID
func (r *gormTodoRepository) DeleteTodo(id uint) error {
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

// func GetAllTodos() ([]model.Todo, error) {
// 	rows, err := config.DB.Query("SELECT id, title , completed FROM todos")
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()
// 	var todos []model.Todo
// 	for rows.Next() {
// 		var todo model.Todo
// 		if err := rows.Scan(&todo.ID, &todo.Title, &todo.Completed); err != nil {
// 			return nil, err
// 		}
// 		todos = append(todos, todo)
// 	}
// 	return todos, nil
// }

// func AddTodo(todo model.Todo) (int64, error) {
// 	result, err := config.DB.Exec("INSERT INTO todos (title, completed) VALUES (?, ?)", todo.Title, todo.Completed)

// 	if err != nil {
// 		log.Println("Error inserting todo:", err)
// 	}
// 	id, err := result.LastInsertId()
// 	if err != nil {
// 		return 0, err
// 	}
// 	return id, nil
// }

// func GetOneTodo(id int) (model.Todo, error) {
// 	var todo model.Todo
// 	err := config.DB.QueryRow("select id, title, completed from todos where id = ?", id).
// 		Scan(&todo.ID, &todo.Title, &todo.Completed)
// 	if err != nil {
// 		log.Println("Error fetching todo:", err)
// 		return model.Todo{}, err
// 	}
// 	return todo, nil
// }
// func UpdateTodo(id int, todo model.Todo) error {
// 	_, err := config.DB.Exec("update todos set title = ?, completed = ? where id = ?", todo.Title, todo.Completed, id)

// 	return err

// }

// func DeleteTodo(id int) error {
// 	_, err := config.DB.Exec("delete from todos where id =?", id)
// 	return err
// }
