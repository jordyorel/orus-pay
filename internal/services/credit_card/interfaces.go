package credit_card

import (
	"context"
	"orus/internal/models"
)

type Service interface {
	ValidateCard(ctx context.Context, userID, cardID uint) error
	GetCard(ctx context.Context, cardID uint) (*models.CreditCard, error)
	TokenizeCreditCard(card models.CreateCardInput) (*models.VisaCardToken, error)
	LinkCard(userID uint, input models.CreateCardInput) (*models.CreditCard, error)
}
