package repository

import (
	"log"

	"github.com/codetheuri/todolist/config"
	"github.com/codetheuri/todolist/internal/app/models"
)

func GetAllTodos() ([]model.Todo, error) {
	rows, err := config.DB.Query("SELECT id, title , completed FROM todos")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var todos []model.Todo
	for rows.Next() {
		var todo model.Todo
		if err := rows.Scan(&todo.ID, &todo.Title, &todo.Completed); err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}
	return todos, nil
}

func AddTodo(todo model.Todo) (int64, error) {
	result, err := config.DB.Exec("INSERT INTO todos (title, completed) VALUES (?, ?)", todo.Title, todo.Completed)

	if err != nil {
		log.Println("Error inserting todo:", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func GetOneTodo(id int) (model.Todo, error) {
	var todo model.Todo
	err := config.DB.QueryRow("select id, title, completed from todos where id = ?", id).
		Scan(&todo.ID, &todo.Title, &todo.Completed)
	if err != nil {
		log.Println("Error fetching todo:", err)
		return model.Todo{}, err
	}
	return todo, nil
}
func UpdateTodo(id int, todo model.Todo) error {
	_, err := config.DB.Exec("update todos set title = ?, completed = ? where id = ?", todo.Title, todo.Completed, id)
	
	return err

}

func DeleteTodo(id int) error {
	_, err := config.DB.Exec("delete from todos where id =?", id)
	return err
}
