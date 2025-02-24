package transaction

import (
	"context"
	"orus/internal/models"
)

// Service defines the main transaction service interface
type Service interface {
	// Core transaction methods
	ProcessTransaction(ctx context.Context, tx *models.Transaction) (*models.Transaction, error)
	ProcessP2PTransfer(ctx context.Context, req TransferRequest) (*models.Transaction, error)
	GetTransaction(ctx context.Context, id string) (*models.Transaction, error)

	// Batch operations
	ProcessBatchTransactions(ctx context.Context, txs []*models.Transaction) ([]*models.Transaction, error)

	// Validation and status
	ValidateTransaction(ctx context.Context, tx *models.Transaction) error
	GetTransactionStatus(ctx context.Context, id string) (string, error)
}

// TransactionProcessor handles the core transaction processing
type TransactionProcessor interface {
	Process(ctx context.Context, tx *models.Transaction) error
	Rollback(ctx context.Context, tx *models.Transaction) error
}

// WalletOperator handles wallet operations for transactions
type WalletOperator interface {
	Debit(ctx context.Context, userID uint, amount float64) error
	Credit(ctx context.Context, userID uint, amount float64) error
	ValidateBalance(ctx context.Context, userID uint, amount float64) error
}
