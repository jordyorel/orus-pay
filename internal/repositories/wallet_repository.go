package repositories

import (
	"context"
	"errors"
	"orus/internal/models"
	"time"
)

var (
	ErrWalletNotFound     = errors.New("wallet not found")
	ErrInvalidWalletData  = errors.New("invalid wallet data")
	ErrDuplicateWallet    = errors.New("wallet already exists")
	ErrTransactionFailed  = errors.New("transaction failed")
	ErrInvalidTransaction = errors.New("invalid transaction")
)

// WalletRepository defines the interface for wallet-related database operations
type WalletRepository interface {
	// Core wallet operations
	Create(wallet *models.Wallet) error
	GetByID(id uint) (*models.Wallet, error)
	GetByUserID(userID uint) (*models.Wallet, error)
	Update(wallet *models.Wallet) error
	Delete(id uint) error

	// Transaction operations
	CreateTransaction(tx *models.Transaction) error
	GetTransactionByID(id uint) (*models.Transaction, error)
	GetTransactionHistory(ctx context.Context, walletID uint, limit, offset int, dest interface{}) error
	GetDailyTransactionTotal(ctx context.Context, userID uint, start, end time.Time, txType string, total *float64) error
	GetMonthlyTransactionTotal(ctx context.Context, userID uint, start, end time.Time, txType string, total *float64) error

	// Batch operations
	ExecuteInTransaction(fn func(WalletRepository) error) error
	BulkCreate(wallets []*models.Wallet) error
	BulkUpdate(wallets []*models.Wallet) error

	// Status operations
	UpdateStatus(walletID uint, status string) error
	GetWalletsByStatus(status string) ([]*models.Wallet, error)

	// Analytics and reporting
	GetTotalBalance() (float64, error)
	GetActiveWalletsCount() (int64, error)
	GetTransactionStats(start, end time.Time) (*TransactionStats, error)
}

// TransactionStats represents aggregated transaction statistics
type TransactionStats struct {
	TotalTransactions int64
	TotalVolume       float64
	AvgAmount         float64
	MaxAmount         float64
	MinAmount         float64
	SuccessRate       float64
}
