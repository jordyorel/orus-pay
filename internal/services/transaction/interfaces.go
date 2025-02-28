package transaction

import (
	"context"
	"orus/internal/models"
)

type WalletService interface {
	Process(ctx context.Context, tx *models.Transaction) error
	Rollback(ctx context.Context, tx *models.Transaction) error
	Debit(ctx context.Context, userID uint, amount float64) error
	Credit(ctx context.Context, userID uint, amount float64) error
	UpdateBalanceOnly(ctx context.Context, userID uint, amount float64) error
}

type BalanceService interface {
	ValidateBalance(ctx context.Context, userID uint, amount float64) error
}

type TransferRequest struct {
	SenderID    uint                   `json:"-"` // Set by handler
	ReceiverID  uint                   `json:"receiver_id"`
	Amount      float64                `json:"amount"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type Service interface {
	Process(ctx context.Context, tx *models.Transaction) error
	Rollback(ctx context.Context, tx *models.Transaction) error
	CreateTransaction(ctx context.Context, tx *models.Transaction) (*models.Transaction, error)
	ProcessTransaction(ctx context.Context, tx *models.Transaction) (*models.Transaction, error)
}
