package creditcard

import (
	"orus/internal/models"
)

// CreateCardInput represents the input for creating a new card
type CreateCardInput struct {
	CardNumber  string `json:"card_number"`
	ExpiryMonth string `json:"expiry_month"`
	ExpiryYear  string `json:"expiry_year"`
}

// TokenizedCard represents a tokenized credit card
type TokenizedCard struct {
	Token    string
	CardType string
	LastFour string
	IssuedBy string
}

// Service defines the interface for credit card operations
type Service interface {
	LinkCard(userID uint, input CreateCardInput) (*models.CreditCard, error)
	GetUserCards(userID uint) ([]models.CreditCard, error)
	DeleteCard(userID uint, cardID uint) error
	GetByID(cardID uint) (*models.CreditCard, error)
}
