package models

import "gorm.io/gorm"

type Todo struct {
	gorm.Model
	Title string `json:"title" gorm:"not null"`
	Description string `json:"description"`
	Completed  bool  `json:"completed" gorm:"default:false"`
}
