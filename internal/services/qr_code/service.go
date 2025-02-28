package qr_code

import (
	"context"
	"errors"
	"fmt"
	domainQR "orus/internal/domain/qr"
	appErrors "orus/internal/errors"
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
		UserType:     user.Role, // Set the user type
		Metadata: models.NewJSON(map[string]interface{}{
			"qr_type":   "receive",
			"user_id":   userID,
			"user_type": user.Role,
			"user_role": user.Role,
		}),
	}

	if err := s.db.Create(qr).Error; err != nil {
		return nil, fmt.Errorf("failed to create QR code: %w", err)
	}

	return qr, nil
}

func (s *service) GetUserPaymentCodeQR(ctx context.Context, userID uint) (*models.QRCode, error) {
	// Get user type first
	user, err := repositories.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	qr := &models.QRCode{
		UserID:    userID,
		Code:      utils.MustGenerateSecureCode(),
		Type:      string(TypePaymentCode),
		Status:    "active",
		ExpiresAt: nil,       // Static codes don't expire
		MaxUses:   -1,        // Unlimited uses
		UserType:  user.Role, // Set the user type
		Metadata: models.NewJSON(map[string]interface{}{
			"qr_type":   "payment_code",
			"user_id":   userID,
			"user_type": user.Role,
			"user_role": user.Role,
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
		return nil, appErrors.ErrQRExpired
	}

	// Check scanner role and QR type validity
	isMerchant := false
	if meta, ok := metadata["scanner_role"].(string); ok && meta == "merchant" {
		isMerchant = true
	}

	// Validate QR type based on scanner role
	if isMerchant {
		// Merchants should scan customer's payment code QR
		if qr.Type != string(TypePaymentCode) {
			return nil, fmt.Errorf("merchants can only scan customer payment code QRs")
		}
	} else {
		// Regular users should scan receive QR codes
		if qr.Type != string(TypeReceive) {
			return nil, fmt.Errorf("users can only scan receive QRs")
		}
	}

	// Create transaction record
	tx := &models.Transaction{
		Type:          getTransactionType(isMerchant),
		SenderID:      getSenderID(isMerchant, qr.UserID, scannerID),
		ReceiverID:    getReceiverID(isMerchant, qr.UserID, scannerID),
		Amount:        amount,
		Status:        "completed",
		Description:   description,
		TransactionID: fmt.Sprintf("QR-%d-%d", scannerID, time.Now().UnixNano()),
		Reference:     fmt.Sprintf("QRP-%d-%d", scannerID, time.Now().UnixNano()),
		PaymentType:   "qr_scan",
		PaymentMethod: "wallet",
		Category:      "Payment",
		MerchantID:    getMerchantID(isMerchant, scannerID),
		Metadata:      models.NewJSON(metadata),
	}

	// Use transaction service to handle the entire operation
	return s.transactionSvc.ProcessTransaction(ctx, tx)
}

func (s *service) ValidateQRCode(ctx context.Context, code string, amount float64) (uint, error) {
	// Get QR code from database
	var qrCode models.QRCode
	err := s.db.Where("code = ? AND status = ?", code, "active").First(&qrCode).Error
	if err != nil {
		return 0, fmt.Errorf("invalid QR code: %w", err)
	}

	// Check if QR code is valid
	if qrCode.ExpiresAt != nil && qrCode.ExpiresAt.Before(time.Now()) {
		return 0, errors.New("QR code expired")
	}

	// Return the user ID associated with the QR code
	return qrCode.UserID, nil
}

func getTransactionType(isMerchant bool) string {
	if isMerchant {
		return "merchant_scan"
	}
	return "QR_PAYMENT"
}

func getMerchantID(isMerchant bool, scannerID uint) *uint {
	if isMerchant {
		return &scannerID
	}
	return nil
}

func getSenderID(isMerchant bool, qrUserID, scannerID uint) uint {
	if isMerchant {
		return qrUserID
	}
	return scannerID
}

func getReceiverID(isMerchant bool, qrUserID, scannerID uint) uint {
	if isMerchant {
		return scannerID
	}
	return qrUserID
}
