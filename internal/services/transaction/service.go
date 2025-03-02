package transaction

import (
	"context"
	"errors"
	"fmt"
	"orus/internal/models"
	"orus/internal/repositories"
	"time"

	"gorm.io/gorm"
)

var (
	ErrHighRiskTransaction = errors.New("transaction risk too high")
	highRiskThreshold      = 0.8
	ErrInsufficientBalance = errors.New("insufficient balance")
)

type service struct {
	db             *gorm.DB
	walletService  WalletService
	balanceService BalanceService
	cache          repositories.CacheRepository
	riskService    *RiskService
}

func NewService(
	db *gorm.DB,
	walletSvc WalletService,
	balanceSvc BalanceService,
	cache repositories.CacheRepository,
) Service {
	return &service{
		db:             db,
		walletService:  walletSvc,
		balanceService: balanceSvc,
		cache:          cache,
		riskService:    NewRiskService(),
	}
}

func (s *service) ProcessTransaction(ctx context.Context, tx *models.Transaction) (*models.Transaction, error) {
	fmt.Printf("Processing transaction: %+v\n", tx)

	// Validate transaction
	if err := s.validateTransaction(tx); err != nil {
		return nil, err
	}

	// Process in a single database transaction
	err := s.db.Transaction(func(dbTx *gorm.DB) error {
		// Get wallets directly from database to avoid cache issues
		var sourceWallet, destWallet models.Wallet

		if err := repositories.DB.Where("user_id = ?", tx.SenderID).First(&sourceWallet).Error; err != nil {
			fmt.Printf("Source wallet lookup failed: %v\n", err)
			return fmt.Errorf("source wallet not found: %w", err)
		}

		if err := repositories.DB.Where("user_id = ?", tx.ReceiverID).First(&destWallet).Error; err != nil {
			fmt.Printf("Destination wallet lookup failed: %v\n", err)
			return fmt.Errorf("destination wallet not found: %w", err)
		}

		// Verify sufficient balance
		if sourceWallet.Balance < tx.Amount {
			return ErrInsufficientBalance
		}

		// Update balances directly
		sourceWallet.Balance -= tx.Amount
		if err := dbTx.Save(&sourceWallet).Error; err != nil {
			return err
		}

		destWallet.Balance += tx.Amount
		if err := dbTx.Save(&destWallet).Error; err != nil {
			return err
		}

		// Update transaction status
		tx.Status = "completed"
		tx.ProcessedAt = time.Now()

		// Create the transaction record
		return dbTx.Create(tx).Error
	})

	if err != nil {
		fmt.Printf("Transaction failed: %v\n", err)
		return nil, err
	}

	// Invalidate caches for both wallets
	s.cache.DeleteWallet(ctx, tx.SenderID)
	s.cache.DeleteWallet(ctx, tx.ReceiverID)

	return tx, nil
}

func (s *service) Process(ctx context.Context, tx *models.Transaction) error {
	if tx.Type == "debit" {
		return s.walletService.Process(ctx, tx)
	}
	return s.walletService.Process(ctx, tx)
}

func (s *service) Rollback(ctx context.Context, tx *models.Transaction) error {
	return s.walletService.Rollback(ctx, tx)
}

func (s *service) CreateTransaction(ctx context.Context, tx *models.Transaction) (*models.Transaction, error) {
	// Validate transaction
	if tx.Amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}

	if tx.SenderID == 0 && tx.ReceiverID == 0 {
		return nil, errors.New("transaction must have at least one party")
	}

	// Set default values if not provided
	if tx.Status == "" {
		tx.Status = "pending"
	}

	// Save to database
	if err := s.db.Create(tx).Error; err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return tx, nil
}

func (s *service) validateTransaction(tx *models.Transaction) error {
	if tx.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if tx.SenderID == 0 && tx.ReceiverID == 0 {
		return errors.New("transaction must have at least one party")
	}
	// Risk assessment
	riskScore := s.riskService.AssessTransaction(tx)
	if riskScore > highRiskThreshold {
		return ErrHighRiskTransaction
	}
	return nil
}

type RiskService struct{}

func NewRiskService() *RiskService {
	return &RiskService{}
}

func (s *RiskService) AssessTransaction(tx *models.Transaction) float64 {
	var riskScore float64 = 0.0
	if tx.Amount > 10000 {
		riskScore += 0.3
	}
	return riskScore
}
