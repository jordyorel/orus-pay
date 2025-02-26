package wallet

import (
	"context"
	"orus/internal/models"
	"time"
)

// TransferRequest represents a wallet transfer request
type TransferRequest struct {
	FromWalletID uint
	ToWalletID   uint
	Amount       float64
	Description  string
	Metadata     map[string]interface{}
}

// WalletConfig holds configuration for wallet operations
type WalletConfig struct {
	DefaultCurrency   string
	MaxDailyLimit     float64
	MaxMonthlyLimit   float64
	MinBalance        float64
	Limits            map[string]TransactionLimits
	WithdrawalFees    map[string]float64
	ProcessingTimeout time.Duration
}

// TransactionLimits defines limits based on user role
type TransactionLimits struct {
	MaxTransactionAmount  float64
	DailyTransactionLimit float64
	MonthlyLimit          float64
	MinTransactionAmount  float64
}

// OperationRequest represents a wallet operation request
type OperationRequest struct {
	WalletID  uint
	Amount    float64
	Type      string // "credit" or "debit"
	Reference string
	Metadata  map[string]interface{}
}

// TransactionHistory represents a wallet transaction record
type TransactionHistory struct {
	ID          uint
	WalletID    uint
	Amount      float64
	Type        string
	Balance     float64
	Description string
	CreatedAt   time.Time
}

// MetricsCollector defines the interface for collecting wallet metrics
type MetricsCollector interface {
	// Operation metrics
	RecordOperationDuration(operation string, duration time.Duration)
	RecordOperationResult(operation, result string)

	// Cache metrics
	RecordCacheHit(key string)
	RecordCacheMiss(key string)

	// Balance metrics
	RecordBalanceChange(userID uint, oldBalance, newBalance float64)

	// Error metrics
	RecordError(operation, errType string)

	// Transaction metrics
	RecordTransaction(txType string, amount float64)
	RecordTransactionVolume(amount float64)
	RecordDailyVolume(userID uint, amount float64)
}

// CacheOperator defines the caching operations needed for transactions
type CacheOperator interface {
	GetWallet(ctx context.Context, userID uint) (*models.Wallet, error)
	SetWallet(ctx context.Context, wallet *models.Wallet) error
	InvalidateWallet(ctx context.Context, userID uint) error
	Get(key string) (interface{}, error)
	Set(key string, value interface{}, expiry time.Duration) error
	Delete(key string) error
}

type contextKey string

const (
	UserRoleContextKey contextKey = "userRole"
)
