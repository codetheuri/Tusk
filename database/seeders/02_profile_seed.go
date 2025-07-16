package seeders

import (
	"log"

	"github.com/codetheuri/todolist/internal/app/models"
	"gorm.io/gorm"
)

type ProfileTableSeeder struct{}

func (s ProfileTableSeeder) Name() string {
	return "02ProfileTableSeeder"
}
func (s *ProfileTableSeeder) Run(db *gorm.DB) error {
	log.Printf("Running seeder: %s", s.Name())
	profiles := []models.Profile{
		{
			UserID:    1,
			DisplayName:  "Admin User",
			Bio:       "This is the admin profile",
			AvatarURL: "https://example.com/avatar/admin.png",
		},
		{
			UserID:    2,
			DisplayName:  "John Doe",
			Bio:       "This is John Doe's profile",
			AvatarURL: "https://example.com/avatar/johndoe.png",
		},
	}

	for _, profile := range profiles {
		var existingProfile models.Profile
		res := db.Where("user_id = ?", profile.UserID).First(&existingProfile)
		if res.Error == nil {
			log.Printf("Profile for user ID %d already exists, skipping...", profile.UserID)
			continue
		}
		if res.Error != nil && res.Error != gorm.ErrRecordNotFound {
			return res.Error
		}
		if err := db.Create(&profile).Error; err != nil {
			return err
		}
		log.Printf("Seeded profile for user ID: %d", profile.UserID)
	}
	return nil
}

func init() {
	RegisteredSeeders = append(RegisteredSeeders, &ProfileTableSeeder{})
}