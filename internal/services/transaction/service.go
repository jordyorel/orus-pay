package transaction

import (
	"context"
	"fmt"
	"orus/internal/models"
	"time"

	"orus/internal/services/wallet"

	"gorm.io/gorm"
)

type service struct {
	db        *gorm.DB
	processor TransactionProcessor
	wallet    WalletOperator
	cache     wallet.CacheOperator
	config    TransactionConfig
}

// NewService creates a new transaction service
func NewService(db *gorm.DB, processor TransactionProcessor, wallet WalletOperator, cache wallet.CacheOperator) Service {
	if db == nil {
		panic("db is required")
	}
	if processor == nil {
		panic("processor is required")
	}
	if wallet == nil {
		panic("wallet is required")
	}
	if cache == nil {
		panic("cache is required")
	}

	return &service{
		db:        db,
		processor: processor,
		wallet:    wallet,
		cache:     cache,
		config: TransactionConfig{
			MaxAmount:     DefaultMaxAmount,
			MinAmount:     DefaultMinAmount,
			RetryAttempts: DefaultMaxRetries,
		},
	}
}

func (s *service) ProcessTransaction(ctx context.Context, tx *models.Transaction) (*models.Transaction, error) {
	if err := s.ValidateTransaction(ctx, tx); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Start transaction processing
	if err := s.processor.Process(ctx, tx); err != nil {
		return nil, fmt.Errorf("processing failed: %w", err)
	}

	// Cache the result
	cacheKey := fmt.Sprintf("%s%s", TransactionCachePrefix, tx.TransactionID)
	if err := s.cache.Set(cacheKey, tx, time.Hour); err != nil {
		// Log cache error but don't fail the transaction
		fmt.Printf("Failed to cache transaction: %v\n", err)
	}

	return tx, nil
}

func (s *service) ProcessP2PTransfer(ctx context.Context, req TransferRequest) (*models.Transaction, error) {
	tx := &models.Transaction{
		TransactionID: fmt.Sprintf("TX-%d-%d", time.Now().Unix(), req.SenderID),
		Type:          "P2P_TRANSFER",
		SenderID:      req.SenderID,
		ReceiverID:    req.ReceiverID,
		Amount:        req.Amount,
		Description:   req.Description,
		Status:        "completed",
		Currency:      "USD",
		PaymentMethod: "direct",
		PaymentType:   "p2p",
		Metadata:      req.Metadata,
	}

	processed, err := s.ProcessTransaction(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to process transfer: %w", err)
	}

	return processed, nil
}

func (s *service) GetTransaction(ctx context.Context, id string) (*models.Transaction, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("%s%s", TransactionCachePrefix, id)
	if cached, err := s.cache.Get(cacheKey); err == nil {
		if tx, ok := cached.(*models.Transaction); ok {
			return tx, nil
		}
	}

	// Fallback to database
	var tx models.Transaction
	if err := s.db.WithContext(ctx).Where("transaction_id = ?", id).First(&tx).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrTransactionNotFound
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return &tx, nil
}

func (s *service) ProcessBatchTransactions(ctx context.Context, txs []*models.Transaction) ([]*models.Transaction, error) {
	results := make([]*models.Transaction, len(txs))
	for i, tx := range txs {
		processed, err := s.ProcessTransaction(ctx, tx)
		if err != nil {
			return results, fmt.Errorf("failed to process transaction %d: %w", i, err)
		}
		results[i] = processed
	}
	return results, nil
}

func (s *service) ValidateTransaction(ctx context.Context, tx *models.Transaction) error {
	if tx.Amount <= 0 {
		return fmt.Errorf("%w: amount must be positive", ErrInvalidStatus)
	}

	if tx.SenderID == 0 || tx.ReceiverID == 0 {
		return fmt.Errorf("%w: invalid sender or receiver", ErrInvalidStatus)
	}

	// Validate sender's balance
	if err := s.wallet.ValidateBalance(ctx, tx.SenderID, tx.Amount); err != nil {
		return fmt.Errorf("balance validation failed: %w", err)
	}

	return nil
}

func (s *service) GetTransactionStatus(ctx context.Context, id string) (string, error) {
	tx, err := s.GetTransaction(ctx, id)
	if err != nil {
		return "", err
	}
	return tx.Status, nil
}

// Continue with other service methods...
