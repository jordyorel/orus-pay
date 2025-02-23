package services

import (
	"errors"
	"fmt"
	"log"
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/utils"
	"time"

	"gorm.io/gorm"
)

var (
	ErrQRInactive           = errors.New("QR code is not active")
	ErrQRExpired            = errors.New("QR code has expired")
	ErrQRUsageLimit         = errors.New("QR code usage limit exceeded")
	ErrAmountMismatch       = errors.New("amount does not match QR code")
	ErrDailyLimitExceeded   = errors.New("daily transaction limit exceeded")
	ErrMonthlyLimitExceeded = errors.New("monthly transaction limit exceeded")
	ErrUnauthorizedCustomer = errors.New("customer not authorized to use this QR code")
)

type QRService struct {
	transactionService *TransactionService
}

func NewQRService() *QRService {
	return &QRService{
		transactionService: NewTransactionService(),
	}
}

// GenerateQRCode creates QR codes with different rules for regular users and merchants
func (s *QRService) GenerateQRCode(userID uint, userType string, qrType string, amount *float64) (*models.QRCode, error) {
	code, err := utils.GenerateSecureCode()
	if err != nil {
		return nil, err
	}

	qr := &models.QRCode{
		Code:     code,
		UserID:   userID,
		UserType: userType,
		Type:     qrType,
		Amount:   amount,
		Status:   "active",
		MaxUses:  -1, // Default unlimited uses
	}

	// Only set expiration for dynamic QR codes
	if qrType == "dynamic" {
		expires := time.Now().Add(15 * time.Minute)
		qr.ExpiresAt = &expires
		qr.MaxUses = 1 // One-time use for dynamic QR
	}

	// Set limits based on user type
	if userType == "user" {
		dailyLimit := float64(1000)
		monthlyLimit := float64(5000)
		qr.DailyLimit = &dailyLimit
		qr.MonthlyLimit = &monthlyLimit
	} else if userType == "merchant" {
		dailyLimit := float64(10000)
		monthlyLimit := float64(100000)
		qr.DailyLimit = &dailyLimit
		qr.MonthlyLimit = &monthlyLimit
	}

	qr, err = repositories.CreateQRCode(qr)
	if err != nil {
		return nil, err
	}

	return qr, nil
}

// GenerateDynamicQRCode creates a one-time use QR code
func (s *QRService) GenerateDynamicQRCode(userID uint, userType string, amount float64) (*models.QRCode, error) {
	code, err := utils.GenerateSecureCode()
	if err != nil {
		return nil, err
	}

	qr := &models.QRCode{
		Code:     code,
		UserID:   userID,
		UserType: userType,
		Type:     "dynamic",
		Amount:   &amount,
		Status:   "active",
		MaxUses:  1,
	}

	// Set expiration
	expires := time.Now().Add(15 * time.Minute)
	qr.ExpiresAt = &expires

	// Set limits based on user type
	if userType == "merchant" {
		dailyLimit := float64(10000)
		monthlyLimit := float64(100000)
		qr.DailyLimit = &dailyLimit
		qr.MonthlyLimit = &monthlyLimit
	} else {
		dailyLimit := float64(1000)
		monthlyLimit := float64(5000)
		qr.DailyLimit = &dailyLimit
		qr.MonthlyLimit = &monthlyLimit
	}

	qr, err = repositories.CreateQRCode(qr)
	if err != nil {
		return nil, err
	}

	return qr, nil
}

func (s *QRService) ProcessQRPayment(code string, customerID uint, amount float64, metadata map[string]interface{}) (*models.Transaction, error) {
	qr, err := repositories.GetQRCodeByCodeForUpdate(code)
	if err != nil {
		return nil, fmt.Errorf("invalid QR code: %w", err)
	}

	// Determine sender and receiver based on QR type
	var senderID, receiverID uint
	if qr.Type == models.QRTypePayment {
		// Merchant scanned user's payment QR
		senderID = qr.UserID    // User pays
		receiverID = customerID // Merchant receives (customerID is merchant here)
	} else if qr.Type == models.QRTypeStatic {
		// User scanned merchant's static QR
		senderID = customerID  // User pays
		receiverID = qr.UserID // Merchant receives
	} else {
		return nil, fmt.Errorf("unsupported QR type for payment: %s", qr.Type)
	}

	// Validate transaction
	walletService := NewWalletService()
	senderWallet, err := walletService.GetWallet(senderID)
	if err != nil || senderWallet.Balance < amount {
		return nil, errors.New("insufficient balance")
	}

	// Create transaction
	tx := &models.Transaction{
		TransactionID: fmt.Sprintf("TX-%d-%d", time.Now().Unix(), senderID),
		Type:          "QR_PAYMENT",
		SenderID:      senderID,
		ReceiverID:    receiverID,
		Amount:        amount,
		QRCodeID:      qr.Code,
		QRType:        qr.Type,
		QROwnerID:     qr.UserID,
		QROwnerType:   qr.UserType,
		Status:        "pending",
		Metadata:      metadata,
	}

	// Process in DB transaction
	err = repositories.DB.Transaction(func(db *gorm.DB) error {
		if err := walletService.Debit(senderID, amount); err != nil {
			return err
		}
		if err := walletService.Credit(receiverID, amount); err != nil {
			return err
		}
		tx.Status = "completed"
		return db.Create(tx).Error
	})
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (s *QRService) ValidateQRCode(qr *models.QRCode, amount float64) error {
	if qr.Status != "active" {
		return ErrQRInactive
	}

	// For payment QR codes
	if qr.Type == models.QRTypePayment {
		// Check if user has sufficient balance
		wallet, err := repositories.GetWalletByUserID(qr.UserID)
		if err != nil {
			return err
		}
		if wallet.Balance < amount {
			return errors.New("insufficient balance")
		}
		return nil
	}

	return errors.New("invalid QR code type")
}

// Additional helper methods...

func (s *QRService) GeneratePaymentCode(userID uint) (*models.QRCode, error) {
	// Check if user already has a payment code
	existingQR, err := repositories.GetUserPaymentQR(userID)
	if err == nil {
		return existingQR, nil // Return existing QR if found
	}

	// Create new payment QR if not found
	qrCode := &models.QRCode{
		Code:     utils.MustGenerateSecureCode(),
		UserID:   userID,
		UserType: "user",
		Type:     models.QRTypePayment,
		Status:   "active",
		MaxUses:  -1, // Never expires
	}

	qrCode, err = repositories.CreateQRCode(qrCode)
	if err != nil {
		return nil, err
	}

	return qrCode, nil
}

func (s *QRService) ValidatePaymentCode(code string) (*models.QRCode, error) {
	qr, err := repositories.GetQRCodeByCode(code)
	if err != nil {
		return nil, err
	}

	// Validate payment code - check for both payment types
	if qr.Type != models.QRTypePayment && qr.Type != models.QRTypePaymentCode {
		return nil, errors.New("invalid QR code type")
	}

	// Check if active
	if qr.Status != "active" {
		return nil, ErrQRInactive
	}

	// Check if expired (for dynamic codes)
	if qr.ExpiresAt != nil && time.Now().After(*qr.ExpiresAt) {
		return nil, ErrQRExpired
	}

	// Add debug logging
	log.Printf("Validating QR code: %+v", qr)

	return qr, nil
}

func (s *QRService) GeneratePaymentQR(userID uint, amount float64) (*models.QRCode, error) {
	uniqueID, err := utils.GenerateUniqueID(8) // Generate an 8-byte unique ID
	if err != nil {
		return nil, err
	}

	// Create a unique QR Code
	qrCode := &models.QRCode{
		Code:     fmt.Sprintf("QR-%d-%s", userID, uniqueID),
		UserID:   userID,
		UserType: "user",
		Type:     models.QRTypeDynamic,
		Amount:   &amount,
		Status:   "active",
		MaxUses:  1, // One-time use
	}

	// Set expiry (e.g., 15 minutes from now)
	expires := time.Now().Add(15 * time.Minute)
	qrCode.ExpiresAt = &expires

	// Create pending transaction
	tx := &models.Transaction{
		ReceiverID: userID,
		Amount:     amount,
		Status:     "pending",
		QRCodeID:   qrCode.Code,
		Type:       models.TransactionTypeQRPayment,
	}

	// Save both QR and transaction in a DB transaction
	err = repositories.DB.Transaction(func(db *gorm.DB) error {
		if err := db.Create(qrCode).Error; err != nil {
			return err
		}
		return db.Create(tx).Error
	})

	if err != nil {
		return nil, err
	}

	return qrCode, nil
}
