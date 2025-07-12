package migrations

import (
	"log"
	"time"

	"gorm.io/gorm"
)

// users_users struct implements migration interface
type users_users struct {
	ID          uint `gorm:"primarykey"`
	Title       string
	Description string
	Completed   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (m *users_users) Version() string {
	return "20250712140525"
}
func (m *users_users) Name() string {
	return "users"
}

// up migration method
func (m *users_users) Up(tx *gorm.DB) error {
	log.Printf("Running Up migration: users", m.Name())
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
	type users_users struct {
		gorm.Model
		Username string `gorm:"uniqueIndex;not null"`
		Email    string `gorm:"uniqueIndex;not null"`
		Password string `gorm:"not null"`
		Role     string `gorm:"default:'user'"`
	}
	if err := tx.AutoMigrate(&users_users{}); err != nil {
		return err
	}
	log.Printf("Successfully applied Up migration: users", m.Name())
	return nil
}

// down migration method
func (m *users_users) Down(tx *gorm.DB) error {
	log.Printf("Running Down migration: users", m.Name())
	// Example:
	// 1. using SQL calls
	// if err := tx.Exec("DROP TABLE IF EXISTS example").Error; err != nil {
	// 	return err
	// }

	// 2. using gorm methods
	// if err := tx.Migrator().DropTable("new_models"); err != nil {
	// 	return err
	// }
	if err := tx.Migrator().DropTable("users_users"); err != nil {
		return err
	}
	log.Printf("Successfully applied Down migration: users", m.Name())
	return nil
}

func init() {
	// Register the migration
	RegisteredMigrations = append(RegisteredMigrations, &users_users{})
}
