package payment

import (
	"context"
	"orus/internal/models"
)

// Service defines the payment service interface
type Service interface {
	// P2P transfers
	SendMoney(ctx context.Context, senderID, receiverID uint, amount float64, description string) (*models.Transaction, error)

	// QR payments
	ProcessQRPayment(ctx context.Context, qrCode string, amount float64, userID uint, description string, metadata map[string]interface{}) (*models.Transaction, error)

	// Merchant payments
	ProcessMerchantPayment(ctx context.Context, customerID, merchantID uint, amount float64, description string) (*models.Transaction, error)
}

// Dependencies required by the payment service
type WalletService interface {
	GetWallet(ctx context.Context, userID uint) (*models.Wallet, error)
	Credit(ctx context.Context, userID uint, amount float64) error
	Debit(ctx context.Context, userID uint, amount float64) error
	ValidateBalance(ctx context.Context, userID uint, amount float64) error
}

type TransactionService interface {
	CreateTransaction(ctx context.Context, tx *models.Transaction) (*models.Transaction, error)
	ProcessTransaction(ctx context.Context, tx *models.Transaction) (*models.Transaction, error)
}

type QRService interface {
	ValidateQRCode(ctx context.Context, code string, amount float64) (uint, error)
}
