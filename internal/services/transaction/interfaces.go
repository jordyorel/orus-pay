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
}

type BalanceService interface {
	ValidateBalance(ctx context.Context, userID uint, amount float64) error
}

type TransferRequest struct {
	SenderID    uint
	ReceiverID  uint
	Amount      float64
	Description string
	Metadata    map[string]interface{}
}
