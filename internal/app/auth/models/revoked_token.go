package models

import (
	"time"

	"gorm.io/gorm"
)

type RevokedToken struct {
	gorm.Model
	JTI       string    `gorm:"unique;not null;size:255" json:"jti"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
}
