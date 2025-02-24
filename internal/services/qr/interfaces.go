package qr

import (
	"context"
	"orus/internal/models"
	"time"
)

// TransactionProcessor handles transaction processing
type TransactionProcessor interface {
	ProcessTransaction(ctx context.Context, tx *models.Transaction) (*models.Transaction, error)
}

// WalletService handles wallet operations
type WalletService interface {
	GetBalance(ctx context.Context, userID uint) (float64, error)
	Debit(ctx context.Context, userID uint, amount float64) error
	Credit(ctx context.Context, userID uint, amount float64) error
}

// Service defines the interface for QR code operations
type Service interface {
	// Generation methods
	GenerateQRCode(ctx context.Context, userID uint, userType string, qrType string, amount *float64) (*models.QRCode, error)
	GeneratePaymentQR(ctx context.Context, userID uint, amount float64) (*models.QRCode, error)
	GeneratePaymentCode(ctx context.Context, userID uint) (*models.QRCode, error)

	// Processing methods
	ProcessQRPayment(ctx context.Context, code string, amount float64, payerID uint, description string, metadata map[string]interface{}) (*models.Transaction, error)
	ValidateQRCode(ctx context.Context, code string) (*models.QRCode, error)

	// Static QR methods - only these two
	GetUserReceiveQR(ctx context.Context, userID uint) (*models.QRCode, error)
	GetUserPaymentCodeQR(ctx context.Context, userID uint) (*models.QRCode, error)
}

// GenerateQRRequest encapsulates parameters for QR generation
type GenerateQRRequest struct {
	UserID       uint
	UserType     UserType
	QRType       QRType
	Amount       *float64
	ExpiresAt    *time.Time
	MaxUses      int
	DailyLimit   *float64
	MonthlyLimit *float64
	Metadata     map[string]interface{}
}
