package models

import "gorm.io/gorm"

type Transaction struct {
	gorm.Model
	SenderID    uint    `gorm:"not null"`
	ReceiverID  uint    `gorm:"not null"`
	Amount      float64 `gorm:"not null"`
	Status      string  `gorm:"default:'pending'"`
	QRCodeID    string  `gorm:"not null"`
	Type        string  `json:"type"`
	PaymentType string  `json:"payment_type"` // New field
	CardID      *uint   `json:"card_id"`      // New field
	Currency    string  `json:"currency"`
}
