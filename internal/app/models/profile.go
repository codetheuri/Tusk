package models

import "gorm.io/gorm"

type Profile struct {
	gorm.Model
	UserID      uint   `json:"user_id" gorm:"not null;uniqueIndex"`
	DisplayName string `json:"display_name" gorm:"not null"`
	Bio         string `json:"bio"`
	AvatarURL   string `json:"avatar_url"`
	User        User   `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
