package wallet

import (
	"context"
	"fmt"
	"log"
	"orus/internal/models"
	"time"

	"gorm.io/gorm"
)

type service struct {
	db      DB
	cache   CacheOperator
	config  WalletConfig
	metrics MetricsCollector
}

// NewService creates a new wallet service
func NewService(db DB, cache CacheOperator, config WalletConfig, metrics MetricsCollector) Service {
	if db == nil {
		panic("db is required")
	}
	if cache == nil {
		panic("cache is required")
	}

	// Set default configuration values if not provided
	if config.DefaultCurrency == "" {
		config.DefaultCurrency = DefaultCurrency
	}
	if config.MaxDailyLimit == 0 {
		config.MaxDailyLimit = DefaultMaxDailyLimit
	}
	if config.ProcessingTimeout == 0 {
		config.ProcessingTimeout = DefaultTimeout
	}

	// Metrics is optional, create no-op collector if nil
	if metrics == nil {
		metrics = &noopMetricsCollector{}
	}

	return &service{
		db:      db,
		cache:   cache,
		config:  config,
		metrics: metrics,
	}
}

func (s *service) GetWallet(ctx context.Context, userID uint) (*models.Wallet, error) {
	var wallet models.Wallet
	err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("id").
		First(&wallet).Error

	if err == gorm.ErrRecordNotFound {
		// Silently create new wallet - this is expected for new users
		wallet = models.Wallet{
			UserID:   userID,
			Balance:  0,
			Currency: "USD",
			Status:   "active",
		}
		if err := s.db.WithContext(ctx).Create(&wallet).Error; err != nil {
			return nil, fmt.Errorf("failed to create wallet: %w", err)
		}
		return &wallet, nil
	}

	if err != nil {
		log.Printf("Error getting wallet: %v", err) // Only log unexpected errors
		return nil, err
	}

	return &wallet, nil
}

func (s *service) Credit(ctx context.Context, userID uint, amount float64) error {
	start := time.Now()
	defer func() {
		s.metrics.RecordOperationDuration("credit", time.Since(start))
	}()

	if amount <= 0 {
		s.metrics.RecordError("credit", "invalid_amount")
		return ErrInvalidAmount
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var wallet models.Wallet
		if err := tx.WithContext(ctx).Set("gorm:for_update", true).
			Where("user_id = ?", userID).First(&wallet).Error; err != nil {
			s.metrics.RecordError("credit", "wallet_not_found")
			return fmt.Errorf("failed to get wallet: %w", err)
		}

		oldBalance := wallet.Balance
		wallet.Balance += amount
		wallet.UpdatedAt = time.Now()

		if err := tx.Save(&wallet).Error; err != nil {
			s.metrics.RecordError("credit", "save_failed")
			return fmt.Errorf("failed to update wallet: %w", err)
		}

		// Record metrics
		s.metrics.RecordBalanceChange(userID, oldBalance, wallet.Balance)
		s.metrics.RecordTransactionVolume(amount)
		s.metrics.RecordDailyVolume(userID, amount)
		s.metrics.RecordOperationResult("credit", "success")

		// Record the transaction
		if err := s.recordTransaction(tx, userID, amount, "credit", fmt.Sprintf("%s transaction of %.2f", "credit", amount)); err != nil {
			s.metrics.RecordError("credit", "record_transaction_failed")
			return fmt.Errorf("failed to record transaction: %w", err)
		}

		// Invalidate all related caches
		s.invalidateWalletCaches(ctx, userID)

		return nil
	})
}

func (s *service) Debit(ctx context.Context, userID uint, amount float64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var wallet models.Wallet
		if err := tx.WithContext(ctx).Set("gorm:for_update", true).
			Where("user_id = ?", userID).First(&wallet).Error; err != nil {
			return fmt.Errorf("failed to get wallet: %w", err)
		}

		if wallet.Balance < amount {
			return ErrInsufficientBalance
		}

		wallet.Balance -= amount
		wallet.UpdatedAt = time.Now()

		if err := tx.Save(&wallet).Error; err != nil {
			return fmt.Errorf("failed to update wallet: %w", err)
		}

		// Invalidate cache
		s.cache.InvalidateWallet(ctx, userID)

		return nil
	})
}

func (s *service) GetBalance(ctx context.Context, userID uint) (float64, error) {
	wallet, err := s.GetWallet(ctx, userID)
	if err != nil {
		return 0, err
	}
	return wallet.Balance, nil
}

func (s *service) ValidateBalance(ctx context.Context, userID uint, amount float64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	wallet, err := s.GetWallet(ctx, userID)
	if err != nil {
		return err
	}

	if wallet.Balance < amount {
		return ErrInsufficientBalance
	}

	return nil
}

func (s *service) CreateWallet(ctx context.Context, userID uint, currency string) (*models.Wallet, error) {
	// Validate currency
	if currency == "" {
		currency = s.config.DefaultCurrency
	}

	// Check if wallet already exists
	existing, err := s.GetWallet(ctx, userID)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("wallet already exists for user %d", userID)
	}

	wallet := &models.Wallet{
		UserID:   userID,
		Balance:  0,
		Currency: currency,
	}

	if err := s.db.WithContext(ctx).Create(wallet).Error; err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	return wallet, nil
}

func (s *service) UpdateWallet(ctx context.Context, wallet *models.Wallet) error {
	if wallet == nil {
		return fmt.Errorf("wallet cannot be nil")
	}

	wallet.UpdatedAt = time.Now()

	if err := s.db.WithContext(ctx).Save(wallet).Error; err != nil {
		return fmt.Errorf("failed to update wallet: %w", err)
	}

	// Invalidate cache
	s.cache.InvalidateWallet(ctx, wallet.UserID)

	return nil
}

func (s *service) ProcessBatchTransfers(ctx context.Context, transfers []TransferRequest) error {
	if len(transfers) == 0 {
		return nil
	}

	type transferResult struct {
		Transfer TransferRequest
		Error    error
	}
	results := make([]transferResult, 0)

	err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, transfer := range transfers {
			// Validate transfer
			if err := s.validateTransfer(ctx, transfer); err != nil {
				results = append(results, transferResult{Transfer: transfer, Error: err})
				continue
			}

			// Check limits
			if err := s.checkDailyLimit(ctx, transfer.FromWalletID, transfer.Amount); err != nil {
				results = append(results, transferResult{Transfer: transfer, Error: err})
				continue
			}

			if err := s.checkMonthlyLimit(ctx, transfer.FromWalletID, transfer.Amount); err != nil {
				results = append(results, transferResult{Transfer: transfer, Error: err})
				continue
			}

			// Process transfer
			if err := s.processTransfer(ctx, tx, transfer); err != nil {
				results = append(results, transferResult{Transfer: transfer, Error: err})
				continue
			}

			results = append(results, transferResult{Transfer: transfer, Error: nil})
		}

		// If any transfer failed, rollback the entire batch
		for _, result := range results {
			if result.Error != nil {
				return fmt.Errorf("batch transfer failed: %w", result.Error)
			}
		}

		return nil
	})

	if err != nil {
		// Log detailed results for debugging
		for _, result := range results {
			if result.Error != nil {
				s.metrics.RecordError("batch_transfer", "failed")
				fmt.Printf("Transfer from %d to %d failed: %v\n",
					result.Transfer.FromWalletID,
					result.Transfer.ToWalletID,
					result.Error)
			}
		}
		return err
	}

	return nil
}

// Helper methods

func (s *service) validateWalletStatus(wallet *models.Wallet) error {
	if wallet == nil {
		return fmt.Errorf("wallet is nil")
	}

	// For now, just check if wallet exists
	// TODO: Add status field to wallet model if needed
	return nil
}

func (s *service) checkDailyLimit(ctx context.Context, userID uint, amount float64) error {
	// Get today's start and end time in UTC
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	var dailyTotal float64
	err := s.db.WithContext(ctx).
		Table("wallet_transactions").
		Where("wallet_id = ? AND created_at BETWEEN ? AND ? AND type = ?",
			userID, startOfDay, endOfDay, "debit").
		Select("COALESCE(SUM(amount), 0)").
		Scan(&dailyTotal).Error

	if err != nil {
		return fmt.Errorf("failed to check daily limit: %w", err)
	}

	if dailyTotal+amount > s.config.MaxDailyLimit {
		return ErrDailyLimitExceeded
	}

	return nil
}

func (s *service) checkMonthlyLimit(ctx context.Context, userID uint, amount float64) error {
	// Get current month's start and end time in UTC
	now := time.Now().UTC()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	var monthlyTotal float64
	err := s.db.WithContext(ctx).
		Table("wallet_transactions").
		Where("wallet_id = ? AND created_at BETWEEN ? AND ? AND type = ?",
			userID, startOfMonth, endOfMonth, "debit").
		Select("COALESCE(SUM(amount), 0)").
		Scan(&monthlyTotal).Error

	if err != nil {
		return fmt.Errorf("failed to check monthly limit: %w", err)
	}

	if monthlyTotal+amount > s.config.MaxMonthlyLimit {
		return ErrMonthlyLimitExceeded
	}

	return nil
}

func (s *service) GetTransactionHistory(ctx context.Context, walletID uint, limit, offset int) ([]TransactionHistory, error) {
	cacheKey := fmt.Sprintf("%s:history:%d:%d", WalletCachePrefix, walletID, limit)

	// Try cache first for common queries
	if offset == 0 && (limit == 10 || limit == 20) {
		if cached, err := s.cache.Get(cacheKey); err == nil {
			if history, ok := cached.([]TransactionHistory); ok {
				return history, nil
			}
		}
	}

	var history []TransactionHistory
	err := s.db.WithContext(ctx).
		Table("wallet_transactions").
		Where("wallet_id = ?", walletID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&history).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get transaction history: %w", err)
	}

	// Cache the result for common queries
	if offset == 0 && (limit == 10 || limit == 20) {
		if err := s.cache.Set(cacheKey, history, CacheDuration); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Failed to cache transaction history: %v\n", err)
		}
	}

	return history, nil
}

func (s *service) recordTransaction(tx *gorm.DB, walletID uint, amount float64, txType string, description string) error {
	transaction := &models.Transaction{
		Type:        txType,
		Amount:      amount,
		Status:      "completed",
		Description: description,
		SenderID:    walletID,
		ReceiverID:  walletID,
	}
	return tx.Create(transaction).Error
}

// Add new cache invalidation helper
func (s *service) invalidateWalletCaches(ctx context.Context, userID uint) {
	s.cache.InvalidateWallet(ctx, userID)

	keys := []string{
		fmt.Sprintf("%s:balance:%d", WalletCachePrefix, userID),
		fmt.Sprintf("%s:history:%d", WalletCachePrefix, userID),
	}

	for _, key := range keys {
		if err := s.cache.Delete(key); err != nil {
			fmt.Printf("Failed to invalidate cache key %s: %v\n", key, err)
		}
	}
}

// Add helper method for transfer validation
func (s *service) validateTransfer(ctx context.Context, transfer TransferRequest) error {
	if transfer.Amount <= 0 {
		return ErrInvalidAmount
	}

	if transfer.FromWalletID == 0 || transfer.ToWalletID == 0 {
		return fmt.Errorf("%w: invalid wallet IDs", ErrInvalidOperation)
	}

	// Get and validate source wallet
	sourceWallet, err := s.GetWallet(ctx, transfer.FromWalletID)
	if err != nil {
		return err
	}
	if err := s.validateWalletStatus(sourceWallet); err != nil {
		return err
	}

	// Get and validate destination wallet
	destWallet, err := s.GetWallet(ctx, transfer.ToWalletID)
	if err != nil {
		return err
	}
	if err := s.validateWalletStatus(destWallet); err != nil {
		return err
	}

	return nil
}

// Add helper method for processing individual transfers
func (s *service) processTransfer(ctx context.Context, tx *gorm.DB, transfer TransferRequest) error {
	// Debit from source wallet
	if err := s.Debit(ctx, transfer.FromWalletID, transfer.Amount); err != nil {
		return fmt.Errorf("failed to debit from wallet %d: %w", transfer.FromWalletID, err)
	}

	// Credit to destination wallet
	if err := s.Credit(ctx, transfer.ToWalletID, transfer.Amount); err != nil {
		// Rollback the debit if credit fails
		if rbErr := s.Credit(ctx, transfer.FromWalletID, transfer.Amount); rbErr != nil {
			return fmt.Errorf("critical error: credit failed and rollback failed: %v, %v", err, rbErr)
		}
		return fmt.Errorf("failed to credit to wallet %d: %w", transfer.ToWalletID, err)
	}

	// Record the transfer
	if err := s.recordTransaction(tx, transfer.FromWalletID, transfer.Amount, "debit", fmt.Sprintf("%s transaction of %.2f", "debit", transfer.Amount)); err != nil {
		return err
	}
	if err := s.recordTransaction(tx, transfer.ToWalletID, transfer.Amount, "credit", fmt.Sprintf("%s transaction of %.2f", "credit", transfer.Amount)); err != nil {
		return err
	}

	return nil
}

// Process implements TransactionProcessor interface
func (s *service) Process(ctx context.Context, tx *models.Transaction) error {
	if tx.Type == "debit" {
		return s.Debit(ctx, tx.SenderID, tx.Amount)
	}
	return s.Credit(ctx, tx.ReceiverID, tx.Amount)
}

// Rollback implements TransactionProcessor interface
func (s *service) Rollback(ctx context.Context, tx *models.Transaction) error {
	// Reverse the transaction
	if tx.Type == "debit" {
		return s.Credit(ctx, tx.SenderID, tx.Amount)
	}
	return s.Debit(ctx, tx.ReceiverID, tx.Amount)
}

// Add no-op metrics collector
type noopMetricsCollector struct{}

func (n *noopMetricsCollector) RecordOperationDuration(string, time.Duration) {}
func (n *noopMetricsCollector) RecordOperationResult(string, string)          {}
func (n *noopMetricsCollector) RecordCacheHit(string)                         {}
func (n *noopMetricsCollector) RecordCacheMiss(string)                        {}
func (n *noopMetricsCollector) RecordBalanceChange(uint, float64, float64)    {}
func (n *noopMetricsCollector) RecordError(string, string)                    {}
func (n *noopMetricsCollector) RecordTransactionVolume(float64)               {}
func (n *noopMetricsCollector) RecordDailyVolume(uint, float64)               {}
