package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Email        string `gorm:"uniqueIndex;not null"` // Unique index on Email
	Password     string `gorm:"not null"`
	Phone        string `gorm:"uniqueIndex"` // Unique index on Phone
	Role         string `gorm:"default:'user'"`
	TokenVersion int    `gorm:"default:1"`
}
