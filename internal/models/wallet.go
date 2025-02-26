package models

import (
	"time"

	"gorm.io/gorm"
)

type Wallet struct {
	ID           uint    `gorm:"primarykey"`
	UserID       uint    `gorm:"uniqueIndex;not null"`
	Balance      float64 `gorm:"default:0"`
	Currency     string  `gorm:"default:'USD'"`
	Status       string  `gorm:"default:'active'"`
	StatusReason string  `gorm:"default:''"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (w *Wallet) BeforeCreate(tx *gorm.DB) error {
	// Ensure balance starts at 0
	w.Balance = 0.0
	return nil
}
