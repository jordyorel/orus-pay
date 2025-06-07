package transfer

import (
	"context"
	"orus/internal/models"
)

// WalletService defines the wallet operations used by the transfer service.
type WalletService interface {
	ValidateBalance(ctx context.Context, userID uint, amount float64) error
	Debit(ctx context.Context, userID uint, amount float64) error
	Credit(ctx context.Context, userID uint, amount float64) error
}

// NotificationService is used to notify users about transfers.
type NotificationService interface {
	SendTransferNotification(ctx context.Context, userID uint, tx *models.Transaction) error
}

// Service handles P2P money transfers between users.
type Service interface {
	Transfer(ctx context.Context, senderID, receiverID uint, amount float64, description string) (*models.Transaction, error)
}
