package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Email    string `gorm:"unique;not null" json:"email" validate:"required,email"`
	Password string `gorm:"not null" json:"-" validate:"required,min=8"`
	Role     string `gorm:"not null;default:'user'" json:"role"`
}
