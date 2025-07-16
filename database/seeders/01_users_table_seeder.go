package seeders

import (
	"log"

	"github.com/codetheuri/todolist/internal/app/models"
	"gorm.io/gorm"
)

type UsersTableSeeder struct{}

func (s UsersTableSeeder) Name() string {
	return "01UsersTableSeeder"
}

func (s *UsersTableSeeder) Run(db *gorm.DB) error {
	log.Printf("Running seeder: %s", s.Name())
	users := []models.User{

		{
			Username: "admin",
			Email:    "admin@example.com",
			Password: "password123",
			Role:    "admin"	,
			// CreatedAt: time.Now(),
			// UpdatedAt: time.Now(),
		},
		{
			Username: "johndoe",
			Email:    "john.doe@example.com",
			Password: "password123",
			// CreatedAt: time.Now(),
			// UpdatedAt: time.Now(),
		},
	}

	for _, user := range users {
		var existingUser models.User
		res := db.Where("email = ?", user.Email).First(&existingUser)
		if res.Error == nil {
			log.Printf("User with email %s already exists, skipping...", user.Email)
			continue
		}
		if res.Error != nil && res.Error != gorm.ErrRecordNotFound {
			return res.Error
		}
		if err := db.Create(&user).Error; err != nil {
			return err
		}
		log.Printf("Seeded user: %s", user.Username)

	}
	return nil
}
func init() {
	RegisteredSeeders = append(RegisteredSeeders, &UsersTableSeeder{})

}
