package notification

import (
	"context"
	"log"
	"orus/internal/models"
)

// Service is a minimal notification service implementation.
type Service struct{}

// NewService creates a new notification service.
func NewService() *Service { return &Service{} }

// SendTransferNotification logs a transfer notification.
func (s *Service) SendTransferNotification(ctx context.Context, userID uint, tx *models.Transaction) error {
	log.Printf("Notify user %d of transfer %s", userID, tx.TransactionID)
	return nil
}
