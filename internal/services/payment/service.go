package payment

import (
	"context"
	"errors"
	"fmt"
	"orus/internal/models"
	"orus/internal/repositories"
	"time"
)

type service struct {
	walletService      WalletService
	transactionService TransactionService
	qrService          QRService
}

// NewService creates a new payment service
func NewService(
	walletSvc WalletService,
	txSvc TransactionService,
	qrSvc QRService,
) Service {
	return &service{
		walletService:      walletSvc,
		transactionService: txSvc,
		qrService:          qrSvc,
	}
}

// SendMoney handles P2P transfers between users
func (s *service) SendMoney(
	ctx context.Context,
	senderID, receiverID uint,
	amount float64,
	description string,
) (*models.Transaction, error) {
	// Validate the transfer
	if senderID == receiverID {
		return nil, errors.New("cannot transfer to self")
	}

	if amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}

	// Validate sender has sufficient balance
	if err := s.walletService.ValidateBalance(ctx, senderID, amount); err != nil {
		return nil, fmt.Errorf("insufficient balance: %w", err)
	}

	// Create transaction record
	tx := &models.Transaction{
		Type:        "transfer",
		SenderID:    senderID,
		ReceiverID:  receiverID,
		Amount:      amount,
		Description: description,
		Status:      "pending",
	}

	// Process the transaction
	return s.transactionService.ProcessTransaction(ctx, tx)
}

// ProcessQRPayment handles payments via QR code
func (s *service) ProcessQRPayment(
	ctx context.Context,
	qrCode string,
	amount float64,
	userID uint,
	description string,
	metadata map[string]interface{},
) (*models.Transaction, error) {
	// Validate QR code and get receiver ID
	receiverID, err := s.qrService.ValidateQRCode(ctx, qrCode, amount)
	if err != nil {
		return nil, fmt.Errorf("invalid QR code: %w", err)
	}

	// Create transaction with QR metadata
	tx := &models.Transaction{
		Type:        "qr_payment",
		SenderID:    userID,
		ReceiverID:  receiverID,
		Amount:      amount,
		Description: description,
		Status:      "pending",
		Metadata:    models.NewJSON(metadata),
	}

	// Process the transaction
	return s.transactionService.ProcessTransaction(ctx, tx)
}

// ProcessMerchantPayment handles direct merchant payments
func (s *service) ProcessMerchantPayment(
	ctx context.Context,
	customerID, merchantID uint,
	amount float64,
	description string,
) (*models.Transaction, error) {
	// Validate customer has sufficient balance
	if err := s.walletService.ValidateBalance(ctx, customerID, amount); err != nil {
		return nil, fmt.Errorf("insufficient balance: %w", err)
	}

	// Get merchant details
	merchant, err := repositories.GetMerchantByUserID(merchantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get merchant details: %w", err)
	}

	// Create merchant payment transaction
	tx := &models.Transaction{
		Type:             "merchant_payment",
		SenderID:         customerID,
		ReceiverID:       merchantID,
		Amount:           amount,
		Description:      description,
		Status:           "pending",
		TransactionID:    fmt.Sprintf("MTXN-%d-%d", merchantID, time.Now().UnixNano()),
		Reference:        fmt.Sprintf("MREF-%d-%d", merchantID, time.Now().UnixNano()),
		PaymentType:      "qr_scan",
		PaymentMethod:    "wallet",
		MerchantID:       &merchantID,
		Category:         "Sale",
		MerchantName:     merchant.BusinessName,
		MerchantCategory: merchant.BusinessType,
	}

	// Process the transaction
	return s.transactionService.ProcessTransaction(ctx, tx)
}
