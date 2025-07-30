package migrations

import (
	"log"

	"github.com/codetheuri/todolist/internal/app/todo/models"
	"gorm.io/gorm"
)

// Createtodostable struct implements migration interface
type Createtodostable struct{}

func (m *Createtodostable) Version() string {
	return "20250712155753"
}
func (m *Createtodostable) Name() string {
	return "create_todos_table"
}

// up migration method
func (m *Createtodostable) Up(tx *gorm.DB) error {
	log.Printf("Running Up migration: %s", m.Name())
	// Add your migration logic here
	//example :
	// 1. using SQL calls
	// if  err := tx.Exec("CREATE TABLE IF NOT EXISTS example (id INT PRIMARY KEY)").Error; err != nil {
	// 	return err
	// }

	// 2. using gorm methods
	//type NewModel struct {
	// gorm.Model
	// Field1 string
	//}
	// if err := tx.AutoMigrate(&NewModel{}); err != nil {
	// 	return err
	// }
	if err := tx.AutoMigrate(&models.Todo{}); err != nil {
		return err
	}
	log.Printf("Successfully applied Up migration: %s", m.Name())
	return nil
}

// down migration method
func (m *Createtodostable) Down(tx *gorm.DB) error {
	log.Printf("Running Down migration: %s", m.Name())
	// Example:
	// 1. using SQL calls
	// if err := tx.Exec("DROP TABLE IF EXISTS example").Error; err != nil {
	// 	return err
	// }

	// 2. using gorm methods
	// if err := tx.Migrator().DropTable("new_models"); err != nil {
	// 	return err
	// }
	if err := tx.Migrator().DropTable(&models.Todo{}); err != nil {
		return err
	}
	log.Printf("Successfully applied Down migration: %s", m.Name())
	return nil
}

func init() {
	// Register the migration
	RegisteredMigrations = append(RegisteredMigrations, &Createtodostable{})
}
