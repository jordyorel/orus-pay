package models

import "gorm.io/gorm"

type Wallet struct {
	gorm.Model
	UserID   uint    `gorm:"uniqueIndex;not null"`
	Balance  float64 `gorm:"default:0"`
	Currency string  `gorm:"default:'USD'"`
}

func (w *Wallet) BeforeCreate(tx *gorm.DB) error {
	// Ensure balance starts at 0
	w.Balance = 0.0
	return nil
}
