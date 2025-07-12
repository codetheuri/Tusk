package migrations

import (
	"log"

	"github.com/codetheuri/todolist/internal/app/models"
	"gorm.io/gorm"
)

// Createuserstable struct implements migration interface
		type Createuserstable struct {}

		func (m *Createuserstable) Version() string{
			return "20250712170020"
			}
		func (m *Createuserstable) Name() string {
			return "create_users_table"
		}	
			//up migration method
		func (m *Createuserstable) Up(tx *gorm.DB) error {
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
		if err := tx.AutoMigrate(&models.User{}); err != nil {
			return err
		}
		if err := tx.AutoMigrate(&models.Profile{}); err != nil {
			return err
		}
		log.Printf("Successfully applied Up migration: %s", m.Name())
		return nil
		}
		//down migration method
		func (m *Createuserstable) Down(tx *gorm.DB) error {
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
		if err := tx.Migrator().DropTable(&models.User{}, &models.Profile{}); err != nil {
			return err
		}
		if err := tx.Migrator().DropTable(&models.Profile{}); err != nil {
			return err
		}
		log.Printf("Successfully applied Down migration: %s", m.Name())
		return nil
		}

		func init() {
		  // Register the migration
		  RegisteredMigrations = append(RegisteredMigrations, &Createuserstable{})
		}
