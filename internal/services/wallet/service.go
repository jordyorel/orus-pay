package wallet

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/repositories/cache"
	creditcard "orus/internal/services/credit-card"
	"time"
)

type service struct {
	repo        repositories.WalletRepository
	cache       *cache.CacheService
	cardService creditcard.Service
	config      WalletConfig
	metrics     MetricsCollector
}

// NewService creates a new wallet service
func NewService(
	repo repositories.WalletRepository,
	cache *cache.CacheService,
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
	// For critical operations like withdrawals, get fresh data from DB
	if ctx.Value("critical_operation") != nil {
		var wallet models.Wallet
		if err := repositories.DB.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
			return nil, err
		}
		return &wallet, nil
	}

	// Try to get from cache first
	cacheKey := s.cache.GenerateKey("wallet", "user", userID)

	// Check if we have it in cache
	var cachedWallet *models.Wallet
	if _, err := s.cache.Get(ctx, cacheKey, &cachedWallet); err == nil && cachedWallet != nil {
		return cachedWallet, nil
	}

	// If not in cache, get from database
	wallet, err := s.repo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	// Store in cache for future use (with a shorter TTL to reduce stale data issues)
	if err := s.cache.SetWithTTL(ctx, cacheKey, wallet, 1*time.Minute); err != nil {
		log.Printf("Failed to cache wallet: %v", err)
	}

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
	key := s.cache.GenerateKey("wallet", "user", userID)
	s.cache.SetWithTTL(ctx, key, wallet, CacheDuration)
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
	senderKey := s.cache.GenerateKey("wallet", "user", walletID)
	s.cache.Delete(ctx, senderKey)

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
	senderKey := s.cache.GenerateKey("wallet", "user", walletID)
	s.cache.Delete(ctx, senderKey)

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
	senderKey := s.cache.GenerateKey("wallet", "user", wallet.UserID)
	s.cache.Delete(ctx, senderKey)

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
				s.metrics.RecordError("batch_transfer", result.Error.Error())
				fmt.Printf("Transfer from %d to %d failed: %v\n",
					result.Transfer.FromWalletID,
					result.Transfer.ToWalletID,
					result.Error)
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

func (s *service) GetTransactionHistory(ctx context.Context, userID uint, limit, offset int) ([]TransactionHistory, error) {
	// Generate cache key for common queries
	cacheKey := fmt.Sprintf("tx_history:%d:%d:%d", userID, limit, offset)

	// Try cache first for common queries
	if offset == 0 && (limit == 10 || limit == 20) {
		var history []TransactionHistory
		found, _ := s.cache.Get(ctx, cacheKey, &history)
		if found {
			return history, nil
		}
	}

	// Cache miss, fetch from database
	var history []TransactionHistory
	err := s.repo.GetTransactionHistory(ctx, userID, limit, offset, &history)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction history: %w", err)
	}

	// Cache the result for common queries
	if offset == 0 && (limit == 10 || limit == 20) {
		if err := s.cache.SetWithTTL(ctx, cacheKey, history, CacheDuration); err != nil {
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
func (s *service) invalidateWalletCaches(ctx context.Context, userIDs ...uint) {
	for _, userID := range userIDs {
		key := s.cache.GenerateKey("wallet", "user", userID)
		if err := s.cache.Delete(ctx, key); err != nil {
			log.Printf("Failed to invalidate wallet cache for user %d: %v", userID, err)
		} else {
			log.Printf("Cache INVALIDATE: wallet:%d", userID)
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

func (s *service) Transfer(ctx context.Context, fromUserID, toUserID uint, amount float64, description string) (*models.Transaction, error) {
	// Debug logs
	log.Printf("Transfer request - From User: %d, To User: %d, Amount: %.2f\n", fromUserID, toUserID, amount)

	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	if fromUserID == toUserID {
		log.Printf("Transfer blocked - Same user IDs: %d\n", fromUserID)
		return nil, errors.New("cannot transfer to self")
	}

	// Get source wallet directly from database to avoid cache issues
	var sourceWallet models.Wallet
	if err := repositories.DB.Where("user_id = ?", fromUserID).First(&sourceWallet).Error; err != nil {
		log.Printf("Source wallet error - User ID: %d, Error: %v\n", fromUserID, err)
		return nil, fmt.Errorf("source wallet not found: %w", err)
	}

	// Get destination wallet directly from database
	var destWallet models.Wallet
	if err := repositories.DB.Where("user_id = ?", toUserID).First(&destWallet).Error; err != nil {
		log.Printf("Destination wallet error - User ID: %d, Error: %v\n", toUserID, err)
		return nil, fmt.Errorf("destination wallet not found: %w", err)
	}

	log.Printf("Wallets found - Source User: %d (Balance: %.2f), Dest User: %d (Balance: %.2f)\n",
		sourceWallet.UserID, sourceWallet.Balance, destWallet.UserID, destWallet.Balance)

	if sourceWallet.Status != "active" {
		return nil, ErrWalletLocked
	}
	if sourceWallet.Balance < amount {
		return nil, ErrInsufficientBalance
	}

	var transaction *models.Transaction

	// Execute transfer in a transaction
	err := s.repo.ExecuteInTransaction(func(tx repositories.WalletRepository) error {
		// Debit source wallet
		sourceWallet.Balance -= amount
		if err := tx.Update(&sourceWallet); err != nil {
			return err
		}

		// Credit destination wallet
		destWallet.Balance += amount
		if err := tx.Update(&destWallet); err != nil {
			return err
		}

		// Record transaction
		transferTx := &models.Transaction{
			SenderID:      fromUserID,
			ReceiverID:    toUserID,
			Amount:        amount,
			Type:          "transfer",
			Status:        "completed",
			Description:   description,
			TransactionID: fmt.Sprintf("TRF-%d-%d-%d", fromUserID, toUserID, time.Now().UnixNano()),
		}

		if err := tx.CreateTransaction(transferTx); err != nil {
			return err
		}

		transaction = transferTx
		return nil
	})

	if err != nil {
		s.metrics.RecordError("transfer", err.Error())
		return nil, ErrTransactionFailed
	}

	// Invalidate caches
	senderKey := s.cache.GenerateKey("wallet", "user", fromUserID)
	receiverKey := s.cache.GenerateKey("wallet", "user", toUserID)
	s.cache.Delete(ctx, senderKey)
	s.cache.Delete(ctx, receiverKey)
	s.invalidateWalletCaches(ctx, fromUserID, toUserID)

	// Record metrics
	s.metrics.RecordTransaction("transfer", amount)

	return transaction, nil
}

func (s *service) TopUp(ctx context.Context, userID, cardID uint, amount float64) error {
	// Get user role from context
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

	// Get wallet by user ID instead of wallet ID
	wallet, err := s.repo.GetByUserID(userID)
	if err != nil {
		// If wallet not found, create a new one
		if err == repositories.ErrWalletNotFound {
			wallet, err = s.CreateWallet(ctx, userID, "USD")
			if err != nil {
				return fmt.Errorf("failed to create wallet: %w", err)
			}
		} else {
			return fmt.Errorf("wallet not found: %w", err)
		}
	}

	if wallet.Status != "active" {
		return ErrWalletLocked
	}

	// Get card details
	card, err := s.cardService.GetByID(cardID)
	if err != nil {
		return fmt.Errorf("failed to get card details: %w", err)
	}

	// Verify card ownership
	if card.UserID != userID {
		return fmt.Errorf("card does not belong to user")
	}

	cardLastFour := card.CardNumber[len(card.CardNumber)-4:]

	// Process top-up
	err = s.repo.ExecuteInTransaction(func(tx repositories.WalletRepository) error {
		// Round the balance to 2 decimal places when updating
		wallet.Balance = math.Round((wallet.Balance+amount)*100) / 100
		if err := tx.Update(wallet); err != nil {
			return err
		}

		topUpTx := &models.Transaction{
			Type:          "top_up",
			SenderID:      userID,
			ReceiverID:    0, // No receiver for top-ups
			Amount:        amount,
			Status:        "completed",
			TransactionID: fmt.Sprintf("TXN-%d-%d", userID, time.Now().UnixNano()),
			Reference:     fmt.Sprintf("TOP-%d-%d", userID, time.Now().UnixNano()),
			PaymentType:   "card_topup",
			PaymentMethod: "credit_card",
			CardID:        &cardID,
			Category:      "Top Up",
			Description:   fmt.Sprintf("Top up from card ending in %s", cardLastFour),
			Metadata: models.NewJSON(map[string]interface{}{
				"card_last_four": cardLastFour,
				"card_type":      card.CardType,
			}),
		}
		return tx.CreateTransaction(topUpTx)
	})

	if err != nil {
		s.metrics.RecordError("top_up", err.Error())
		return ErrTransactionFailed
	}

	// Invalidate all caches
	senderKey := s.cache.GenerateKey("wallet", "user", userID)
	s.cache.Delete(ctx, senderKey)
	s.invalidateWalletCaches(ctx, userID)

	s.metrics.RecordTransaction("top_up", amount)

	return nil
}

func (s *service) Withdraw(ctx context.Context, userID uint, cardID uint, amount float64) error {
	// Add card validation
	card, err := s.cardService.GetByIDAndUserID(cardID, userID)
	if err != nil {
		return fmt.Errorf("invalid card: %w", err)
	}

	if card.Status != "active" {
		return errors.New("card is not active")
	}

	// Get user role from context
	roleVal := ctx.Value(UserRoleContextKey)
	role, ok := roleVal.(string)
	if !ok || role == "" {
		role = "user" // Default to user fees
	}

	// Calculate fee based on role (keeping your original logic)
	feePercent := s.config.WithdrawalFees[role]
	fee := math.Round(amount*feePercent*100) / 100    // Round fee to 2 decimals
	totalAmount := math.Round((amount+fee)*100) / 100 // Round total to 2 decimals

	if amount <= 0 {
		return ErrInvalidAmount
	}

	// Get fresh wallet data directly from database, bypassing cache
	var wallet models.Wallet
	if err := repositories.DB.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return fmt.Errorf("wallet not found: %w", err)
	}

	if wallet.Balance < totalAmount {
		return ErrInsufficientBalance
	}

	if wallet.Status != "active" {
		return ErrWalletLocked
	}

	err = s.repo.ExecuteInTransaction(func(tx repositories.WalletRepository) error {
		// Round the balance to 2 decimal places when updating
		wallet.Balance = math.Round((wallet.Balance-totalAmount)*100) / 100
		if err := tx.Update(&wallet); err != nil {
			return err
		}

		// Record main withdrawal
		if err := tx.CreateTransaction(&models.Transaction{
			SenderID:    wallet.ID,
			Amount:      amount,
			Type:        "withdrawal",
			Status:      "completed",
			Description: fmt.Sprintf("Withdrawal to card ending in %d", cardID),
			Metadata: models.NewJSON(map[string]any{
				"card_id": cardID,
				"fee":     fee,
			}),
		}); err != nil {
			return err
		}

		// Record fee transaction if there is a fee
		if fee > 0 {
			if err := tx.CreateTransaction(&models.Transaction{
				SenderID:    wallet.ID,
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

	// Invalidate all caches
	senderKey := s.cache.GenerateKey("wallet", "user", userID)
	s.cache.Delete(ctx, senderKey)
	s.invalidateWalletCaches(ctx, userID)

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

	senderKey := s.cache.GenerateKey("wallet", "user", walletID)
	s.cache.Delete(ctx, senderKey)
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

	senderKey := s.cache.GenerateKey("wallet", "user", walletID)
	s.cache.Delete(ctx, senderKey)
	return nil
}

func (s *service) GetWithdrawalFeePercent() float64 {
	// Default to user fee if no role specified
	return s.config.WithdrawalFees["user"]
}

// UpdateBalanceOnly updates a wallet balance directly, bypassing cache
func (s *service) UpdateBalanceOnly(ctx context.Context, userID uint, amount float64) error {
	// Log the operation
	fmt.Printf("Updating balance for user %d by %.2f\n", userID, amount)

	// Get wallet directly from database to avoid cache issues
	var wallet models.Wallet
	if err := repositories.DB.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		fmt.Printf("Failed to find wallet for user %d: %v\n", userID, err)
		return fmt.Errorf("wallet not found: %w", err)
	}

	fmt.Printf("Found wallet ID %d for user %d with current balance %.2f\n",
		wallet.ID, userID, wallet.Balance)

	// Update balance
	wallet.Balance += amount

	// Save directly to database
	if err := repositories.DB.Save(&wallet).Error; err != nil {
		fmt.Printf("Failed to update wallet balance: %v\n", err)
		return err
	}

	fmt.Printf("Updated wallet ID %d for user %d to new balance %.2f\n",
		wallet.ID, userID, wallet.Balance)

	// Invalidate all caches
	senderKey := s.cache.GenerateKey("wallet", "user", userID)
	s.cache.Delete(ctx, senderKey)
	s.invalidateWalletCaches(ctx, userID)

	return nil
}

func (s *service) ClearCache(ctx context.Context, userID uint) error {
	senderKey := s.cache.GenerateKey("wallet", "user", userID)
	return s.cache.Delete(ctx, senderKey)
}
