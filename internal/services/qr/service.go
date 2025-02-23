package qr

import (
	"context"
	"errors"
	"fmt"
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/utils"
	"time"

	"gorm.io/gorm"
)

var (
	ErrInvalidQRType   = errors.New("invalid QR code type")
	ErrInvalidUserType = errors.New("invalid user type")
	ErrQRExpired       = errors.New("QR code has expired")
	ErrQRInactive      = errors.New("QR code is not active")
	ErrQRLimitExceeded = errors.New("QR code usage limit exceeded")
	ErrInvalidAmount   = errors.New("invalid amount")
)

type Service struct {
	db     *gorm.DB
	cache  repositories.CacheRepository
	txProc TransactionProcessor
}

type TransactionProcessor interface {
	ProcessQRTransaction(context.Context, *models.Transaction) error
}

func NewService(db *gorm.DB, cache repositories.CacheRepository, txProc TransactionProcessor) *Service {
	return &Service{
		db:     db,
		cache:  cache,
		txProc: txProc,
	}
}

func (s *Service) GenerateQR(ctx context.Context, config QRConfig) (*models.QRCode, error) {
	if err := s.validateConfig(config); err != nil {
		return nil, err
	}

	code, err := utils.GenerateSecureCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR code: %w", err)
	}

	limits := s.getLimits(config)
	qr := &models.QRCode{
		Code:         code,
		UserID:       config.UserID,
		UserType:     string(config.UserType),
		Type:         string(config.Type),
		Amount:       config.Amount,
		ExpiresAt:    limits.ExpiresAt,
		MaxUses:      limits.MaxUses,
		Status:       "active",
		DailyLimit:   &limits.DailyLimit,
		MonthlyLimit: &limits.MonthlyLimit,
		Metadata:     config.Metadata,
	}

	if err := s.db.Create(qr).Error; err != nil {
		return nil, fmt.Errorf("failed to save QR code: %w", err)
	}

	return qr, nil
}

func (s *Service) ProcessQRPayment(ctx context.Context, code string, amount float64, payer uint) (*models.Transaction, error) {
	qr, err := s.validateAndLockQR(ctx, code)
	if err != nil {
		return nil, err
	}

	// Validate amount for dynamic QR codes
	if qr.Amount != nil && *qr.Amount != amount {
		return nil, ErrInvalidAmount
	}

	// Check limits
	if err := s.checkLimits(ctx, qr, amount); err != nil {
		return nil, err
	}

	// Create transaction
	tx := &models.Transaction{
		TransactionID: fmt.Sprintf("QR-%d-%d", time.Now().Unix(), payer),
		Type:          models.TransactionTypeQRPayment,
		SenderID:      payer,
		ReceiverID:    qr.UserID,
		Amount:        amount,
		QRCodeID:      qr.Code,
		QRType:        qr.Type,
		QROwnerID:     qr.UserID,
		QROwnerType:   qr.UserType,
		Status:        "pending",
	}

	// Process the transaction
	if err := s.txProc.ProcessQRTransaction(ctx, tx); err != nil {
		return nil, err
	}

	// Update QR code usage
	if err := s.updateQRUsage(ctx, qr); err != nil {
		return nil, err
	}

	return tx, nil
}

func (s *Service) validateConfig(config QRConfig) error {
	switch config.Type {
	case TypeStatic, TypeDynamic, TypePayment:
	default:
		return ErrInvalidQRType
	}

	switch config.UserType {
	case UserTypeRegular, UserTypeMerchant:
	default:
		return ErrInvalidUserType
	}

	if config.Type == TypeDynamic && (config.Amount == nil || *config.Amount <= 0) {
		return ErrInvalidAmount
	}

	return nil
}

func (s *Service) getLimits(config QRConfig) QRLimits {
	limits := defaultLimits[config.UserType]

	if config.Type == TypeDynamic {
		expires := time.Now().Add(15 * time.Minute)
		limits.ExpiresAt = &expires
		limits.MaxUses = 1
	}

	if config.DailyLimit != nil {
		limits.DailyLimit = *config.DailyLimit
	}
	if config.MonthlyLimit != nil {
		limits.MonthlyLimit = *config.MonthlyLimit
	}

	return limits
}

// Additional helper methods...
