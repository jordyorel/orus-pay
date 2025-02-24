package models

import "gorm.io/gorm"

// CreateCreditCard represents the user's credit card data (only storing the tokenized values)
type CreateCreditCard struct {
	gorm.Model
	UserID      uint   `gorm:"not null;index" json:"user_id"`
	CardNumber  string `gorm:"not null" json:"card_number"`
	CardType    string `gorm:"not null" json:"card_type"`
	ExpiryMonth string `gorm:"not null" json:"expiry_month"`
	ExpiryYear  string `gorm:"not null" json:"expiry_year"`
	Status      string `gorm:"not null;default:'active'" json:"status"`
}

// VisaCardToken represents the card tokenization result returned after tokenizing the card.
type VisaCardToken struct {
	Token    string `json:"token"`     // Tokenized card number returned from the payment processor
	Expiry   string `json:"expiry"`    // Expiry date in MM/YY format
	CardType string `json:"card_type"` // Card type (Visa, Mastercard, etc.)
}
