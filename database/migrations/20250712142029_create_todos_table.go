package migrations
	import (
		"gorm.io/gorm"
		"log"
)
		// create_todos_table_createtodostable struct implements migration interface
		type create_todos_table_createtodostable struct {}

		func (m *create_todos_table_createtodostable) Version() string{
			return "20250712142029"
			}
		func (m *create_todos_table_createtodostable) Name() string {
			return "create_todos_table"
		}	
			//up migration method
		func (m *create_todos_table_createtodostable) Up(tx *gorm.DB) error {
		log.Printf("Running Up migration: create_todos_table", m.Name())
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
		log.Printf("Successfully applied Up migration: createtodostable", m.Name())
		return nil
		}
		//down migration method
		func (m *create_todos_table_createtodostable) Down(tx *gorm.DB) error {
		log.Printf("Running Down migration: create_todos_table", m.Name())
		// Example:
		// 1. using SQL calls
		// if err := tx.Exec("DROP TABLE IF EXISTS example").Error; err != nil {
		// 	return err
		// }

		// 2. using gorm methods
		// if err := tx.Migrator().DropTable("new_models"); err != nil {
		// 	return err
		// }
		log.Printf("Successfully applied Down migration: createtodostable", m.Name())
		return nil
		}

		func init() {
		  // Register the migration
		  RegisteredMigrations = append(RegisteredMigrations, &create_todos_table_createtodostable{})
		}
