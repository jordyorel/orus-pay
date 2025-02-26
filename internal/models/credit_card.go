package models

import "time"

// CreditCard represents a stored credit card
type CreditCard struct {
	ID          uint   `gorm:"primarykey"`
	UserID      uint   `gorm:"not null;index"`
	CardNumber  string `gorm:"not null"`
	CardType    string `gorm:"not null"`
	ExpiryMonth string `gorm:"not null"`
	ExpiryYear  string `gorm:"not null"`
	LastFour    string `gorm:"not null"`
	IsDefault   bool   `gorm:"default:false"`
	Status      string `gorm:"default:'active'"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// VisaCardToken represents the card tokenization result
type VisaCardToken struct {
	Token    string `json:"token"`
	Expiry   string `json:"expiry"`
	CardType string `json:"card_type"`
}

// CreateCardInput represents the input for creating a new card
type CreateCardInput struct {
	CardNumber  string `json:"card_number"`
	ExpiryMonth string `json:"expiry_month"`
	ExpiryYear  string `json:"expiry_year"`
}
