package qr_code

import (
	"context"
	"fmt"
	domainQR "orus/internal/domain/qr"
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/services/transaction"
	"orus/internal/services/wallet"
	"orus/internal/utils"
	"time"

	"gorm.io/gorm"
)

type service struct {
	db             *gorm.DB
	cache          repositories.CacheRepository
	transactionSvc transaction.Service
	walletSvc      wallet.Service
}

func NewService(
	db *gorm.DB,
	cache repositories.CacheRepository,
	txSvc transaction.Service,
	walletSvc wallet.Service,
) Service {
	return &service{
		db:             db,
		cache:          cache,
		transactionSvc: txSvc,
		walletSvc:      walletSvc,
	}
}

func (s *service) GetUserReceiveQR(ctx context.Context, userID uint) (*models.QRCode, error) {
	// Get user type first
	user, err := repositories.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Get limits based on user type
	var limits QRLimits
	if user.Role == "merchant" {
		limits = DefaultLimits[domainQR.UserTypeMerchant]
	} else {
		limits = DefaultLimits[domainQR.UserTypeRegular]
	}

	qr := &models.QRCode{
		UserID:       userID,
		Code:         utils.MustGenerateSecureCode(),
		Type:         string(TypeReceive),
		Status:       "active",
		ExpiresAt:    nil, // Static codes don't expire
		MaxUses:      limits.MaxUses,
		DailyLimit:   &limits.DailyLimit,
		MonthlyLimit: &limits.MonthlyLimit,
		Metadata: models.NewJSON(map[string]interface{}{
			"qr_type":   "receive",
			"user_id":   userID,
			"user_role": user.Role,
		}),
	}

	if err := s.db.Create(qr).Error; err != nil {
		return nil, fmt.Errorf("failed to create QR code: %w", err)
	}

	return qr, nil
}

func (s *service) GetUserPaymentCodeQR(ctx context.Context, userID uint) (*models.QRCode, error) {
	qr := &models.QRCode{
		UserID:    userID,
		Code:      utils.MustGenerateSecureCode(),
		Type:      string(TypePaymentCode),
		Status:    "active",
		ExpiresAt: nil, // Remove expiration for static codes
		MaxUses:   -1,  // Unlimited uses
		Metadata: models.NewJSON(map[string]interface{}{
			"qr_type": "payment_code",
			"user_id": userID,
		}),
	}

	if err := s.db.Create(qr).Error; err != nil {
		return nil, fmt.Errorf("failed to create QR code: %w", err)
	}

	return qr, nil
}

func (s *service) ProcessQRPayment(ctx context.Context, code string, amount float64, scannerID uint, description string, metadata map[string]interface{}) (*models.Transaction, error) {
	// Get QR code from database
	var qr models.QRCode
	if err := s.db.Where("code = ? AND status = ?", code, "active").First(&qr).Error; err != nil {
		return nil, fmt.Errorf("invalid or expired QR code: %w", err)
	}

	// Check expiry only if ExpiresAt is set
	if qr.ExpiresAt != nil && qr.ExpiresAt.Before(time.Now()) {
		return nil, ErrQRExpired
	}

	// Check scanner role
	isMerchant := false
	if meta, ok := metadata["scanner_role"].(string); ok && meta == "merchant" {
		isMerchant = true
	}

	// Validate QR type based on scanner role
	if isMerchant {
		// Merchants can only scan customer payment codes
		if qr.Type != string(TypePaymentCode) {
			return nil, fmt.Errorf("merchants can only scan payment code QRs")
		}
		// For merchant scanning customer payment code:
		// Customer (QR owner) pays, merchant receives
		return s.processTransaction(qr.UserID, scannerID, amount, description, metadata)
	} else {
		// Regular users can only scan receive QRs
		if qr.Type != string(TypeReceive) {
			return nil, fmt.Errorf("users can only scan receive QRs")
		}
		// For user scanning another user/merchant receive QR:
		// Scanner pays, QR owner receives
		return s.processTransaction(scannerID, qr.UserID, amount, description, metadata)
	}
}

// Helper function to process the actual transaction
func (s *service) processTransaction(payerID, receiverID uint, amount float64, description string, metadata map[string]interface{}) (*models.Transaction, error) {
	// Remove expiry check from here since we don't have qr
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	// Create transaction first
	tx := &models.Transaction{
		Type:        "QR_PAYMENT",
		SenderID:    payerID,
		ReceiverID:  receiverID,
		Amount:      amount,
		Status:      "pending",
		Description: description,
		Metadata:    models.NewJSON(metadata),
	}

	// Create the transaction record
	if err := s.db.Create(tx).Error; err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Process the wallet updates
	if err := s.db.Transaction(func(dtx *gorm.DB) error {
		// Update sender's wallet (debit)
		if err := dtx.Exec("UPDATE wallets SET balance = balance - ? WHERE user_id = ?", amount, payerID).Error; err != nil {
			return err
		}

		// Update receiver's wallet (credit)
		if err := dtx.Exec("UPDATE wallets SET balance = balance + ? WHERE user_id = ?", amount, receiverID).Error; err != nil {
			// Rollback will happen automatically
			return err
		}

		// Update transaction status
		if err := dtx.Model(tx).Update("status", "completed").Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		// If the wallet updates fail, mark transaction as failed
		s.db.Model(tx).Update("status", "failed")
		return nil, err
	}

	return tx, nil
}
