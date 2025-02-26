package repositories

import (
	"errors"
	"orus/internal/models"
)

var (
	ErrCardNotFound = errors.New("credit card not found")
)

type CreditCardRepository interface {
	// Core operations
	GetByID(cardID uint) (*models.CreditCard, error)
	Create(card *models.CreditCard) error
	Update(card *models.CreditCard) error
	Delete(cardID uint) error

	// Query operations
	GetByUserID(userID uint) ([]*models.CreditCard, error)
	GetDefaultCard(userID uint) (*models.CreditCard, error)
	GetActiveCards(userID uint) ([]*models.CreditCard, error)

	// Status operations
	UpdateStatus(cardID uint, status string) error
	SetDefault(cardID uint, isDefault bool) error
}
