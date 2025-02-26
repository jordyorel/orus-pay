package wallet

import (
	"context"
	"errors"
	"fmt"
	"orus/internal/models"
	"orus/internal/repositories"
	creditcard "orus/internal/services/credit-card"
	"time"
)

type service struct {
	repo        repositories.WalletRepository
	cache       repositories.CacheRepository
	cardService creditcard.Service
	config      WalletConfig
	metrics     MetricsCollector
}

// NewService creates a new wallet service
func NewService(
	repo repositories.WalletRepository,
	cache repositories.CacheRepository,
	cardService creditcard.Service,
	config WalletConfig,
	metrics MetricsCollector,
) Service {
	if repo == nil {
		panic("repo is required")
	}
	if cache == nil {
		panic("cache is required")
	}
	if cardService == nil {
		panic("card service is required")
	}

	// Set default configuration values if not provided
	if config.DefaultCurrency == "" {
		config.DefaultCurrency = "USD"
	}
	if config.WithdrawalFees == nil {
		config.WithdrawalFees = map[string]float64{
			"user":     0.001, // 1% for regular users
			"merchant": 0.075, // 0.75% for merchants (competitive rate)
		}
	}
	if config.Limits == nil {
		config.Limits = map[string]TransactionLimits{
			"user": {
				MaxTransactionAmount:  50000, // $5,000 per transaction
				DailyTransactionLimit: 50000, // $10,000 per day
				MonthlyLimit:          50000, // $50,000 per month
				MinTransactionAmount:  1,     // $1 minimum
			},
			"merchant": {
				MaxTransactionAmount:  500000,  // $50,000 per transaction
				DailyTransactionLimit: 500000,  // $100,000 per day
				MonthlyLimit:          1000000, // $1,000,000 per month
				MinTransactionAmount:  1,       // $1 minimum
			},
		}
	}
	if config.ProcessingTimeout == 0 {
		config.ProcessingTimeout = DefaultTimeout
	}

	// Metrics is optional, create no-op collector if nil
	if metrics == nil {
		metrics = &NoopMetricsCollector{}
	}

	return &service{
		repo:        repo,
		cache:       cache,
		cardService: cardService,
		config:      config,
		metrics:     metrics,
	}
}

func (s *service) GetWallet(ctx context.Context, userID uint) (*models.Wallet, error) {
	// Try cache first
	if wallet, err := s.cache.GetWallet(ctx, userID); err == nil {
		return wallet, nil
	}

	// Get from database
	wallet, err := s.repo.GetByUserID(userID)
	if err != nil {
		if err == repositories.ErrWalletNotFound {
			return nil, ErrWalletNotFound
		}
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	// Update cache
	s.cache.SetWallet(ctx, userID, wallet)
	return wallet, nil
}

func (s *service) CreateWallet(ctx context.Context, userID uint, currency string) (*models.Wallet, error) {
	wallet := &models.Wallet{
		UserID:   userID,
		Balance:  0,
		Status:   "active",
		Currency: currency,
	}

	if err := s.repo.Create(wallet); err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	// Update cache
	s.cache.SetWallet(ctx, userID, wallet)
	return wallet, nil
}

func (s *service) Credit(ctx context.Context, walletID uint, amount float64) error {
	// Get user role from context with proper type assertion
	roleVal := ctx.Value(UserRoleContextKey)
	role, ok := roleVal.(string)
	if !ok || role == "" {
		role = "user" // Default to user limits
	}

	limits := s.config.Limits[role]
	if amount <= 0 || amount < limits.MinTransactionAmount {
		return ErrInvalidAmount
	}

	if amount > limits.MaxTransactionAmount {
		return fmt.Errorf("amount exceeds maximum limit of %v", limits.MaxTransactionAmount)
	}

	wallet, err := s.repo.GetByID(walletID)
	if err != nil {
		return fmt.Errorf("failed to get wallet: %w", err)
	}

	if wallet.Status != "active" {
		return ErrWalletLocked
	}

	// Perform the credit operation in a transaction
	err = s.repo.ExecuteInTransaction(func(tx repositories.WalletRepository) error {
		wallet.Balance += amount
		if err := tx.Update(wallet); err != nil {
			return err
		}

		// Record the transaction
		txn := &models.Transaction{
			SenderID:    walletID,
			Type:        "credit",
			Amount:      amount,
			Description: "Wallet credit",
			Status:      "completed",
		}
		return tx.CreateTransaction(txn)
	})

	if err != nil {
		s.metrics.RecordError("credit", err.Error())
		return ErrTransactionFailed
	}

	// Invalidate cache
	s.cache.DeleteWallet(ctx, wallet.UserID)

	// Record metrics
	s.metrics.RecordTransaction("credit", amount)

	return nil
}

func (s *service) Debit(ctx context.Context, walletID uint, amount float64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	wallet, err := s.repo.GetByID(walletID)
	if err != nil {
		return fmt.Errorf("failed to get wallet: %w", err)
	}

	if wallet.Balance < amount {
		return ErrInsufficientBalance
	}

	// Perform the debit operation in a transaction
	err = s.repo.ExecuteInTransaction(func(tx repositories.WalletRepository) error {
		wallet.Balance -= amount
		if err := tx.Update(wallet); err != nil {
			return err
		}

		// Record the transaction
		txn := &models.Transaction{
			SenderID:    walletID,
			Type:        "debit",
			Amount:      amount,
			Description: "Wallet debit",
			Status:      "completed",
		}
		return tx.CreateTransaction(txn)
	})

	if err != nil {
		s.metrics.RecordError("debit", err.Error())
		return ErrTransactionFailed
	}

	// Invalidate cache
	s.cache.DeleteWallet(ctx, wallet.UserID)

	// Record metrics
	s.metrics.RecordTransaction("debit", amount)

	return nil
}

func (s *service) GetBalance(ctx context.Context, walletID uint) (float64, error) {
	wallet, err := s.repo.GetByID(walletID)
	if err != nil {
		return 0, fmt.Errorf("failed to get wallet: %w", err)
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

func (s *service) UpdateWallet(ctx context.Context, wallet *models.Wallet) error {
	if wallet == nil {
		return fmt.Errorf("wallet cannot be nil")
	}

	wallet.UpdatedAt = time.Now()

	if err := s.repo.Update(wallet); err != nil {
		return fmt.Errorf("failed to update wallet: %w", err)
	}

	// Invalidate cache
	s.cache.DeleteWallet(ctx, wallet.UserID)

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

	err := s.repo.ExecuteInTransaction(func(tx repositories.WalletRepository) error {
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
				s.metrics.RecordError("batch_transfer", result.Error.Error())
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

func (s *service) checkDailyLimit(ctx context.Context, userID uint, amount float64) error {
	// Get today's start and end time in UTC
	now := time.Now().UTC()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	var dailyTotal float64
	err := s.repo.GetDailyTransactionTotal(ctx, userID, startOfDay, endOfDay, "debit", &dailyTotal)
	if err != nil {
		return fmt.Errorf("failed to check daily limit: %w", err)
	}

	if dailyTotal+amount > s.config.Limits["user"].DailyTransactionLimit {
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
	err := s.repo.GetMonthlyTransactionTotal(ctx, userID, startOfMonth, endOfMonth, "debit", &monthlyTotal)
	if err != nil {
		return fmt.Errorf("failed to check monthly limit: %w", err)
	}

	if monthlyTotal+amount > s.config.Limits["user"].MaxTransactionAmount {
		return ErrMonthlyLimitExceeded
	}

	return nil
}

func (s *service) GetTransactionHistory(ctx context.Context, walletID uint, limit, offset int) ([]TransactionHistory, error) {
	cacheKey := fmt.Sprintf("%s:history:%d:%d", WalletCachePrefix, walletID, limit)

	// Try cache first for common queries
	if offset == 0 && (limit == 10 || limit == 20) {
		if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
			if history, ok := cached.([]TransactionHistory); ok {
				return history, nil
			}
		}
	}

	var history []TransactionHistory
	err := s.repo.GetTransactionHistory(ctx, walletID, limit, offset, &history)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction history: %w", err)
	}

	// Cache the result for common queries
	if offset == 0 && (limit == 10 || limit == 20) {
		if err := s.cache.Set(ctx, cacheKey, history, CacheDuration); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Failed to cache transaction history: %v\n", err)
		}
	}

	return history, nil
}

func (s *service) recordTransaction(tx repositories.WalletRepository, walletID uint, amount float64, txType string, description string) error {
	transaction := &models.Transaction{
		Type:        txType,
		Amount:      amount,
		Status:      "completed",
		Description: description,
		SenderID:    walletID,
		ReceiverID:  walletID,
	}
	return tx.CreateTransaction(transaction)
}

// Add new cache invalidation helper
func (s *service) invalidateWalletCaches(ctx context.Context, userID uint) {
	s.cache.DeleteWallet(ctx, userID)

	keys := []string{
		fmt.Sprintf("%s:balance:%d", WalletCachePrefix, userID),
		fmt.Sprintf("%s:history:%d", WalletCachePrefix, userID),
	}

	for _, key := range keys {
		if err := s.cache.Delete(ctx, key); err != nil {
			fmt.Printf("Failed to invalidate cache key %s: %v\n", key, err)
		}
	}
}

// Add helper method for transfer validation
func (s *service) validateTransfer(ctx context.Context, transfer TransferRequest) error {
	// Get user role from context
	roleVal := ctx.Value(UserRoleContextKey)
	role, ok := roleVal.(string)
	if !ok || role == "" {
		role = "user" // Default to user limits
	}

	// Check transaction limits
	limits := s.config.Limits[role]
	if transfer.Amount > limits.MaxTransactionAmount {
		return fmt.Errorf("amount exceeds maximum limit of %v", limits.MaxTransactionAmount)
	}

	if transfer.FromWalletID == 0 || transfer.ToWalletID == 0 {
		return fmt.Errorf("%w: invalid wallet IDs", ErrInvalidOperation)
	}

	// Just check if wallets exist
	_, err := s.repo.GetByID(transfer.FromWalletID)
	if err != nil {
		return fmt.Errorf("source wallet not found: %w", err)
	}

	_, err = s.repo.GetByID(transfer.ToWalletID)
	if err != nil {
		return fmt.Errorf("destination wallet not found: %w", err)
	}

	if transfer.FromWalletID == transfer.ToWalletID {
		return errors.New("cannot transfer to self")
	}

	return nil
}

// Add helper method for processing individual transfers
func (s *service) processTransfer(ctx context.Context, tx repositories.WalletRepository, transfer TransferRequest) error {
	// Debit from source wallet
	if err := s.Debit(ctx, transfer.FromWalletID, transfer.Amount); err != nil {
		return fmt.Errorf("failed to debit from wallet %d: %w", transfer.FromWalletID, err)
	}

	// Credit to destination wallet
	if err := s.Credit(ctx, transfer.ToWalletID, transfer.Amount); err != nil {
		// Rollback the debit if credit fails
		if rbErr := s.Debit(ctx, transfer.FromWalletID, transfer.Amount); rbErr != nil {
			return fmt.Errorf("critical error: debit failed and rollback failed: %v, %v", err, rbErr)
		}
		return fmt.Errorf("failed to credit to wallet %d: %w", transfer.ToWalletID, err)
	}

	// Record the transfer
	if err := s.recordTransaction(tx, transfer.FromWalletID, transfer.Amount, "debit", transfer.Description); err != nil {
		return err
	}
	if err := s.recordTransaction(tx, transfer.ToWalletID, transfer.Amount, "credit", transfer.Description); err != nil {
		return err
	}

	return nil
}

// Process implements TransactionProcessor interface
func (s *service) Process(ctx context.Context, tx *models.Transaction) error {
	if tx.Type == "debit" {
		return s.Debit(ctx, tx.SenderID, tx.Amount)
	}
	return s.Credit(ctx, tx.SenderID, tx.Amount)
}

// Rollback implements TransactionProcessor interface
func (s *service) Rollback(ctx context.Context, tx *models.Transaction) error {
	// Reverse the transaction
	if tx.Type == "debit" {
		return s.Credit(ctx, tx.SenderID, tx.Amount)
	}
	return s.Debit(ctx, tx.SenderID, tx.Amount)
}

func (s *service) Transfer(ctx context.Context, fromUserID, toUserID uint, amount float64, description string) error {
	// Debug logs
	// fmt.Printf("Transfer request - From User: %d, To User: %d, Amount: %.2f\n", fromUserID, toUserID, amount)

	if amount <= 0 {
		return ErrInvalidAmount
	}

	if fromUserID == toUserID {
		fmt.Printf("Transfer blocked - Same user IDs: %d\n", fromUserID)
		return errors.New("cannot transfer to self")
	}

	// Get source wallet
	sourceWallet, err := s.repo.GetByUserID(fromUserID)
	if err != nil {
		fmt.Printf("Source wallet error - User ID: %d, Error: %v\n", fromUserID, err)
		return fmt.Errorf("source wallet not found: %w", err)
	}

	// Get destination wallet
	destWallet, err := s.repo.GetByUserID(toUserID)
	if err != nil {
		fmt.Printf("Destination wallet error - User ID: %d, Error: %v\n", toUserID, err)
		return fmt.Errorf("destination wallet not found: %w", err)
	}

	fmt.Printf("Wallets found - Source User: %d (Balance: %.2f), Dest User: %d (Balance: %.2f)\n",
		sourceWallet.UserID, sourceWallet.Balance, destWallet.UserID, destWallet.Balance)

	if sourceWallet.Status != "active" {
		return ErrWalletLocked
	}
	if sourceWallet.Balance < amount {
		return ErrInsufficientBalance
	}

	// Execute transfer in a transaction
	err = s.repo.ExecuteInTransaction(func(tx repositories.WalletRepository) error {
		// Debit source wallet
		sourceWallet.Balance -= amount
		if err := tx.Update(sourceWallet); err != nil {
			return err
		}

		// Credit destination wallet
		destWallet.Balance += amount
		if err := tx.Update(destWallet); err != nil {
			return err
		}

		// Record transaction
		transferTx := &models.Transaction{
			SenderID:    fromUserID,
			ReceiverID:  toUserID,
			Amount:      amount,
			Type:        "transfer",
			Status:      "completed",
			Description: description,
		}
		return tx.CreateTransaction(transferTx)
	})

	if err != nil {
		s.metrics.RecordError("transfer", err.Error())
		return ErrTransactionFailed
	}

	// Invalidate caches
	s.invalidateWalletCaches(ctx, sourceWallet.UserID)
	s.invalidateWalletCaches(ctx, destWallet.UserID)

	// Record metrics
	s.metrics.RecordTransaction("transfer", amount)

	return nil
}

func (s *service) TopUp(ctx context.Context, userID uint, cardID uint, amount float64) error {
	// Get user role from context
	roleVal := ctx.Value(UserRoleContextKey)
	role, ok := roleVal.(string)
	if !ok || role == "" {
		role = "user" // Default to user limits
	}

	// Debug log
	// fmt.Printf("TopUp - Role: %s, Amount: %.2f, Limit: %.2f\n", role, amount, s.config.Limits[role].MaxTransactionAmount)

	limits := s.config.Limits[role]
	if amount <= 0 {
		return ErrInvalidAmount
	}

	if amount > limits.MaxTransactionAmount {
		return fmt.Errorf("amount exceeds maximum limit of %v", limits.MaxTransactionAmount)
	}

	wallet, err := s.repo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("wallet not found: %w", err)
	}

	if wallet.Status != "active" {
		return ErrWalletLocked
	}

	// Process top-up
	err = s.repo.ExecuteInTransaction(func(tx repositories.WalletRepository) error {
		wallet.Balance += amount
		if err := tx.Update(wallet); err != nil {
			return err
		}

		topUpTx := &models.Transaction{
			SenderID:    userID,
			Amount:      amount,
			Type:        "top_up",
			Status:      "completed",
			Description: fmt.Sprintf("Top up from card ending in %d", cardID),
			Metadata: models.NewJSON(map[string]interface{}{
				"card_id": cardID,
			}),
		}
		return tx.CreateTransaction(topUpTx)
	})

	if err != nil {
		s.metrics.RecordError("top_up", err.Error())
		return ErrTransactionFailed
	}

	s.invalidateWalletCaches(ctx, wallet.UserID)
	s.metrics.RecordTransaction("top_up", amount)

	return nil
}

func (s *service) Withdraw(ctx context.Context, walletID uint, cardID uint, amount float64) error {
	// Get user role from context
	roleVal := ctx.Value(UserRoleContextKey)
	role, ok := roleVal.(string)
	if !ok || role == "" {
		role = "user" // Default to user fees
	}

	// Calculate fee based on role
	feePercent := s.config.WithdrawalFees[role]
	fee := amount * feePercent

	if amount <= 0 {
		return ErrInvalidAmount
	}

	wallet, err := s.repo.GetByID(walletID)
	if err != nil {
		return fmt.Errorf("wallet not found: %w", err)
	}

	if wallet.Balance < amount {
		return ErrInsufficientBalance
	}

	if wallet.Status != "active" {
		return ErrWalletLocked
	}

	err = s.repo.ExecuteInTransaction(func(tx repositories.WalletRepository) error {
		wallet.Balance -= amount
		if err := tx.Update(wallet); err != nil {
			return err
		}

		// Record main withdrawal
		if err := tx.CreateTransaction(&models.Transaction{
			SenderID:    walletID,
			Amount:      amount,
			Type:        "withdrawal",
			Status:      "completed",
			Description: fmt.Sprintf("Withdrawal to card ending in %d", cardID),
			Metadata: models.NewJSON(map[string]interface{}{
				"card_id": cardID,
				"fee":     fee,
			}),
		}); err != nil {
			return err
		}

		// Record fee transaction if there is a fee
		if fee > 0 {
			if err := tx.CreateTransaction(&models.Transaction{
				SenderID:    walletID,
				Amount:      fee,
				Type:        "fee",
				Status:      "completed",
				Description: "Withdrawal fee",
				Metadata: models.NewJSON(map[string]interface{}{
					"withdrawal_amount": amount,
					"fee_percent":       feePercent,
				}),
			}); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		s.metrics.RecordError("withdrawal", err.Error())
		return ErrTransactionFailed
	}

	s.invalidateWalletCaches(ctx, wallet.UserID)
	s.metrics.RecordTransaction("withdrawal", amount)

	return nil
}

func (s *service) LockWallet(ctx context.Context, walletID uint, reason string) error {
	wallet, err := s.repo.GetByID(walletID)
	if err != nil {
		return fmt.Errorf("wallet not found: %w", err)
	}

	wallet.Status = "locked"
	wallet.StatusReason = reason

	if err := s.repo.Update(wallet); err != nil {
		return fmt.Errorf("failed to lock wallet: %w", err)
	}

	s.invalidateWalletCaches(ctx, wallet.UserID)
	return nil
}

func (s *service) UnlockWallet(ctx context.Context, walletID uint) error {
	wallet, err := s.repo.GetByID(walletID)
	if err != nil {
		return fmt.Errorf("wallet not found: %w", err)
	}

	wallet.Status = "active"
	wallet.StatusReason = ""

	if err := s.repo.Update(wallet); err != nil {
		return fmt.Errorf("failed to unlock wallet: %w", err)
	}

	s.invalidateWalletCaches(ctx, wallet.UserID)
	return nil
}

func (s *service) GetWithdrawalFeePercent() float64 {
	// Default to user fee if no role specified
	return s.config.WithdrawalFees["user"]
}
