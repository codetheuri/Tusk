package migrations

import (
	"log"

	"github.com/codetheuri/todolist/internal/app/auth/models"
	"gorm.io/gorm"
)

// Createauthtables struct implements migration interface
		type Createauthtables struct {}

		func (m *Createauthtables) Version() string{
			return "20250717130322"
			}
		func (m *Createauthtables) Name() string {
			return "create_auth_tables"
		}	
			//up migration method
		func (m *Createauthtables) Up(tx *gorm.DB) error {
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
		if err:= tx.AutoMigrate(&models.User{}); err!= nil {
			return err
		} 
		if err := tx.AutoMigrate(&models.RevokedToken{}); err != nil {
			return err
		}
		log.Printf("Successfully applied Up migration: %s", m.Name())
		return nil
		}
		//down migration method
		func (m *Createauthtables) Down(tx *gorm.DB) error {
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
		if err := tx.Migrator().DropTable(&models.User{}); err != nil {
			return err
		}
		if err := tx.Migrator().DropTable(&models.RevokedToken{}); err != nil {
			return err
		}
		log.Printf("Successfully applied Down migration: %s", m.Name())
		return nil
		}

		func init() {
		  // Register the migration
		  RegisteredMigrations = append(RegisteredMigrations, &Createauthtables{})
		}
