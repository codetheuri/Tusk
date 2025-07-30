package seeders

import (
	"log"

	"github.com/codetheuri/todolist/internal/app/todo/models"
	"gorm.io/gorm"
)

type TodoTableSeeder struct{}

func (s TodoTableSeeder) Name() string {
	return "TodoTableSeeder"
}

func (s *TodoTableSeeder) Run(db *gorm.DB) error {
	log.Printf("Running seeder: %s", s.Name())
	todos := []models.Todo{
		{
			Title:       "Buy groceries",
			Description: "Milk, Bread, Eggs",
			Completed:      false,
		
		},
		{
			Title:       "Complete project report",
			Description: "Finish the report by end of the week",
			Completed:      true,
		
		},
	}

	for _, todo := range todos {
		var existingTodo models.Todo
		res := db.Where("title = ?", todo.Title).First(&existingTodo)
		if res.Error == nil {
			log.Printf("Todo with title %s already exists, skipping...", todo.Title)
			continue
		}
		if res.Error != nil && res.Error != gorm.ErrRecordNotFound {
			return res.Error
		}
		if err := db.Create(&todo).Error; err != nil {
			return err
		}
		log.Printf("Seeded todo: %s", todo.Title)
	}
	return nil
}
func init() {
	RegisteredSeeders = append(RegisteredSeeders, &TodoTableSeeder{})
}
