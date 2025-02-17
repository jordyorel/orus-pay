package models

import "gorm.io/gorm"

// CreateCreditCard represents the user's credit card data (only storing the tokenized values)
type CreateCreditCard struct {
	gorm.Model
	UserID      uint   `gorm:"not null" json:"user_id"`
	CardNumber  string `gorm:"not null" json:"card_number"`
	ExpiryMonth string `gorm:"not null" json:"expiry_month"`
	ExpiryYear  string `gorm:"not null" json:"expiry_year"`
	CardType    string `gorm:"default:'Visa'" json:"card_type"`
	Token       string `json:"token"`
}

// VisaCardToken represents the card tokenization result returned after tokenizing the card.
type VisaCardToken struct {
	Token    string `json:"token"`     // Tokenized card number returned from the payment processor
	Expiry   string `json:"expiry"`    // Expiry date in MM/YY format
	CardType string `json:"card_type"` // Card type (Visa, Mastercard, etc.)
}
