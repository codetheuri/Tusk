package seeders

import "gorm.io/gorm"

type Seeder interface {
	Name() string
	Run(*gorm.DB) error
}

var RegisteredSeeders []Seeder
