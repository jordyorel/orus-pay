package models

import "gorm.io/gorm"

type Wallet struct {
	gorm.Model
	UserID   uint    `gorm:"not null"`
	Balance  float64 `gorm:"default:0.0"`
	Currency string  `gorm:"default:'USD'"`
	QRCodeID string  `gorm:"unique;not null"`
}
