package wallet

import (
	"context"
	"errors"
	"fmt"
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/utils/cache"

	"gorm.io/gorm"
)

var (
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrInvalidAmount       = errors.New("invalid amount")
	ErrWalletNotFound      = errors.New("wallet not found")
)

type Operation string

const (
	OperationCredit Operation = "credit"
	OperationDebit  Operation = "debit"
)

type WalletOperation struct {
	UserID    uint
	Operation Operation
	Amount    float64
	Reference string
	Metadata  map[string]interface{}
}

type WalletService struct {
	db    *gorm.DB
	cache repositories.CacheRepository
}

func NewWalletService(db *gorm.DB, cache repositories.CacheRepository) *WalletService {
	return &WalletService{
		db:    db,
		cache: cache,
	}
}

func (s *WalletService) ProcessOperation(ctx context.Context, op WalletOperation) error {
	if op.Amount <= 0 {
		return ErrInvalidAmount
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		wallet, err := s.getWalletForUpdate(tx, op.UserID)
		if err != nil {
			return err
		}

		switch op.Operation {
		case OperationDebit:
			if wallet.Balance < op.Amount {
				return ErrInsufficientBalance
			}
			wallet.Balance -= op.Amount
		case OperationCredit:
			wallet.Balance += op.Amount
		default:
			return fmt.Errorf("unsupported operation: %s", op.Operation)
		}

		if err := tx.Save(wallet).Error; err != nil {
			return err
		}

		// Record the operation in transaction history
		txn := &models.Transaction{
			TransactionID: op.Reference,
			Type:          string(op.Operation),
			SenderID:      op.UserID,
			ReceiverID:    op.UserID,
			Amount:        op.Amount,
			Status:        "completed",
			Reference:     op.Reference,
			Description:   fmt.Sprintf("%s operation", op.Operation),
			Metadata:      models.NewJSON(op.Metadata),
			Currency:      "USD", // Default currency
		}

		if err := tx.Create(txn).Error; err != nil {
			return err
		}

		// Invalidate cache
		cacheKey := cache.GenerateKey(cache.EntityWallet, cache.KeyID, op.UserID)
		s.cache.Delete(ctx, cacheKey)

		return nil
	})
}

func (s *WalletService) getWalletForUpdate(tx *gorm.DB, userID uint) (*models.Wallet, error) {
	var wallet models.Wallet
	if err := tx.Set("gorm:for_update", true).
		Where("user_id = ?", userID).
		First(&wallet).Error; err != nil {
		return nil, ErrWalletNotFound
	}
	return &wallet, nil
}

func (s *WalletService) GetBalance(ctx context.Context, userID uint) (float64, error) {
	cacheKey := cache.GenerateKey(cache.EntityWallet, cache.KeyID, userID)

	// Try cache first
	if balance, err := s.cache.GetFloat64(ctx, cacheKey); err == nil {
		return balance, nil
	}

	wallet, err := s.getWalletForUpdate(s.db, userID)
	if err != nil {
		return 0, err
	}

	// Cache the balance
	s.cache.SetFloat64(ctx, cacheKey, wallet.Balance, repositories.DefaultExpiration)

	return wallet.Balance, nil
}
