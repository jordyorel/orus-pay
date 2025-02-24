package wallet

import (
	"context"
	"database/sql"
	"orus/internal/models"

	"gorm.io/gorm"
)

// Service defines the main wallet service interface
type Service interface {
	// Core wallet operations
	GetWallet(ctx context.Context, userID uint) (*models.Wallet, error)
	Credit(ctx context.Context, userID uint, amount float64) error
	Debit(ctx context.Context, userID uint, amount float64) error

	// Balance operations
	GetBalance(ctx context.Context, userID uint) (float64, error)
	ValidateBalance(ctx context.Context, userID uint, amount float64) error

	// Wallet management
	CreateWallet(ctx context.Context, userID uint, currency string) (*models.Wallet, error)
	UpdateWallet(ctx context.Context, wallet *models.Wallet) error

	// Batch operations
	ProcessBatchTransfers(ctx context.Context, transfers []TransferRequest) error

	// Transaction processing
	Process(ctx context.Context, tx *models.Transaction) error
	Rollback(ctx context.Context, tx *models.Transaction) error
}

type DB interface {
	First(dest interface{}, conds ...interface{}) *gorm.DB
	Save(value interface{}) *gorm.DB
	Transaction(fc func(tx *gorm.DB) error, opts ...*sql.TxOptions) error
	WithContext(ctx context.Context) *gorm.DB
	Create(value interface{}) *gorm.DB
	Where(query interface{}, args ...interface{}) *gorm.DB
}
