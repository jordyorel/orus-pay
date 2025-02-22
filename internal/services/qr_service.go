package services

import (
	"errors"
	"fmt"
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/utils"
	"time"
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
	}

	// Set limits based on user type
	if qrType == "dynamic" {
		expires := time.Now().Add(15 * time.Minute)
		qr.ExpiresAt = &expires
		qr.MaxUses = 1 // One-time use for dynamic QR
	} else {
		qr.MaxUses = -1 // Unlimited uses for static QR
	}

	// Set basic limits for all users
	if userType == "regular" || userType == "user" {
		dailyLimit := float64(1000)
		monthlyLimit := float64(5000)
		qr.DailyLimit = &dailyLimit
		qr.MonthlyLimit = &monthlyLimit
	}

	// Set merchant-specific limits
	if userType == "merchant" {
		dailyLimit := float64(10000)
		monthlyLimit := float64(100000)
		qr.DailyLimit = &dailyLimit
		qr.MonthlyLimit = &monthlyLimit
	}

	return repositories.CreateQRCode(qr)
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

	return repositories.CreateQRCode(qr)
}

func (s *QRService) ProcessQRPayment(code string, customerID uint, amount float64) (*models.Transaction, error) {
	qr, err := repositories.GetQRCodeByCodeForUpdate(code)
	if err != nil {
		return nil, fmt.Errorf("invalid QR code: %w", err)
	}

	// Prevent self-payment and validate QR code
	if customerID == qr.UserID {
		return nil, errors.New("cannot pay to your own QR code")
	}

	if err := s.validateQRCode(qr, customerID, amount); err != nil {
		return nil, err
	}

	// Create QR transaction record
	qrTransaction := &models.QRTransaction{
		QRCodeID:   qr.ID,
		CustomerID: customerID,
		Amount:     amount,
		Status:     "pending",
	}

	// Start transaction
	tx := repositories.DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer tx.Rollback()

	// Save QR transaction
	if err := tx.Create(qrTransaction).Error; err != nil {
		return nil, err
	}

	// Create and process the payment transaction
	transaction := &models.Transaction{
		TransactionID: fmt.Sprintf("TX-%d-%d", time.Now().Unix(), customerID),
		Type:          "qr_payment",
		SenderID:      customerID,
		ReceiverID:    qr.UserID,
		Amount:        amount,
		Status:        "pending",
		QRCodeID:      qr.Code,
	}

	// Process the transaction
	processedTx, err := s.transactionService.ProcessTransaction(transaction)
	if err != nil {
		return nil, err
	}

	// Update QR transaction with transaction ID and status
	now := time.Now()
	qrTransaction.TransactionID = processedTx.ID
	qrTransaction.Status = "completed"
	qrTransaction.CompletedAt = &now
	if err := tx.Save(qrTransaction).Error; err != nil {
		return nil, err
	}

	// Increment QR code usage count
	qr.UsageCount++
	if err := tx.Save(qr).Error; err != nil {
		return nil, err
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return processedTx, nil
}

func (s *QRService) validateQRCode(qr *models.QRCode, customerID uint, amount float64) error {
	if qr.Status != "active" {
		return ErrQRInactive
	}

	// Check expiration for dynamic QR codes
	if qr.Type == "dynamic" {
		if qr.ExpiresAt != nil && time.Now().After(*qr.ExpiresAt) {
			return ErrQRExpired
		}
	}

	// Check usage limit
	if qr.MaxUses > 0 && qr.UsageCount >= qr.MaxUses {
		return ErrQRUsageLimit
	}

	// For dynamic QR codes, amount must match exactly
	if qr.Type == "dynamic" && qr.Amount != nil && *qr.Amount != amount {
		return ErrAmountMismatch
	}

	// Check daily and monthly limits
	if qr.DailyLimit != nil || qr.MonthlyLimit != nil {
		if err := s.checkTransactionLimits(qr, amount); err != nil {
			return err
		}
	}

	// Check allowed customers for merchant QR codes
	if qr.UserType == "merchant" && len(qr.AllowedCustomers) > 0 {
		if !contains(qr.AllowedCustomers, customerID) {
			return ErrUnauthorizedCustomer
		}
	}

	return nil
}

func (s *QRService) checkTransactionLimits(qr *models.QRCode, amount float64) error {
	if qr.DailyLimit != nil {
		dailyTotal, err := repositories.GetQRCodeDailyTotal(qr.ID)
		if err != nil {
			return err
		}
		if dailyTotal+amount > *qr.DailyLimit {
			return ErrDailyLimitExceeded
		}
	}

	if qr.MonthlyLimit != nil {
		monthlyTotal, err := repositories.GetQRCodeMonthlyTotal(qr.ID)
		if err != nil {
			return err
		}
		if monthlyTotal+amount > *qr.MonthlyLimit {
			return ErrMonthlyLimitExceeded
		}
	}

	return nil
}

func contains(slice []uint, item uint) bool {
	for _, i := range slice {
		if i == item {
			return true
		}
	}
	return false
}

// Additional helper methods...
