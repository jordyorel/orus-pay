package qr

import (
	"context"
	"fmt"
	domainErrors "orus/internal/errors"
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/utils"
	"orus/internal/validation"
	"time"

	domainQR "orus/internal/domain/qr"

	"log"

	"gorm.io/gorm"
)

const (
	DefaultMaxUses = 1
)

type service struct {
	db             *gorm.DB
	cache          repositories.CacheRepository
	transactionSvc TransactionProcessor
	walletSvc      WalletService
}

// NewService creates a new QR service instance
func NewService(db *gorm.DB, cache repositories.CacheRepository,
	txSvc TransactionProcessor, walletSvc WalletService) Service {
	if db == nil {
		panic("db is required")
	}
	if cache == nil {
		panic("cache is required")
	}
	if txSvc == nil {
		panic("transaction service is required")
	}
	if walletSvc == nil {
		panic("wallet service is required")
	}

	return &service{
		db:             db,
		cache:          cache,
		transactionSvc: txSvc,
		walletSvc:      walletSvc,
	}
}

func (s *service) GenerateQRCode(ctx context.Context, userID uint, userType string, qrType string, amount *float64) (*models.QRCode, error) {
	expires := time.Now().Add(15 * time.Minute)
	req := domainQR.GenerateQRRequest{
		UserID:    userID,
		UserType:  domainQR.UserType(userType),
		QRType:    domainQR.QRType(qrType),
		Amount:    amount,
		MaxUses:   1,
		ExpiresAt: &expires,
	}
	return s.generateQRCode(ctx, req)
}

func (s *service) GeneratePaymentQR(ctx context.Context, userID uint, amount float64) (*models.QRCode, error) {
	expires := time.Now().Add(15 * time.Minute)
	req := domainQR.GenerateQRRequest{
		UserID:    userID,
		UserType:  domainQR.UserTypeRegular,
		QRType:    domainQR.TypePayment,
		Amount:    &amount,
		ExpiresAt: &expires,
		MaxUses:   1,
	}

	return s.GenerateQRCode(ctx, req.UserID, req.UserType.String(), req.QRType.String(), req.Amount)
}

func (s *service) GenerateDynamicQR(ctx context.Context, userID uint, amount float64) (*models.QRCode, error) {
	expires := time.Now().Add(15 * time.Minute)
	req := domainQR.GenerateQRRequest{
		UserID:    userID,
		UserType:  domainQR.UserTypeRegular,
		QRType:    domainQR.TypeDynamic,
		Amount:    &amount,
		ExpiresAt: &expires,
		MaxUses:   1,
	}

	return s.GenerateQRCode(ctx, req.UserID, req.UserType.String(), req.QRType.String(), req.Amount)
}

func (s *service) ProcessQRPayment(ctx context.Context, code string, amount float64, payer uint, description string, metadata map[string]interface{}) (*models.Transaction, error) {
	qr, err := s.validateAndLockQR(ctx, code)
	if err != nil {
		return nil, err
	}

	if err := s.validatePayment(qr, amount); err != nil {
		return nil, err
	}

	if err := s.updateQRUsage(ctx, qr); err != nil {
		return nil, err
	}

	metadataJSON := models.JSON(metadata)
	tx := &models.Transaction{
		TransactionID: fmt.Sprintf("QR-%d-%d", time.Now().Unix(), payer),
		Type:          models.TransactionTypeQRPayment,
		SenderID:      payer,
		ReceiverID:    qr.UserID,
		Amount:        amount,
		Description:   description,
		QRCodeID:      &qr.Code,
		Status:        "pending",
		PaymentMethod: "qr",
		PaymentType:   "direct",
		QROwnerID:     qr.UserID,
		QROwnerType:   qr.UserType,
		QRType:        qr.Type,
		Currency:      "USD",
		Metadata:      metadataJSON,
	}

	if err := s.processTransaction(ctx, tx); err != nil {
		return nil, err
	}

	return tx, nil
}

func (s *service) ValidateQRCode(ctx context.Context, code string) (*models.QRCode, error) {
	var qr models.QRCode
	if err := s.db.WithContext(ctx).Where("code = ?", code).First(&qr).Error; err != nil {
		return nil, domainErrors.ErrInvalidQR
	}

	if qr.Status != "active" {
		return nil, domainErrors.ErrQRInactive
	}

	if qr.ExpiresAt != nil && time.Now().After(*qr.ExpiresAt) {
		return nil, domainErrors.ErrQRExpired
	}

	return &qr, nil
}

func (s *service) GeneratePaymentCode(ctx context.Context, userID uint) (*models.QRCode, error) {
	req := domainQR.GenerateQRRequest{
		UserID:   userID,
		UserType: domainQR.UserTypeRegular,
		QRType:   domainQR.TypePayment,
	}
	return s.GenerateQRCode(ctx, req.UserID, req.UserType.String(), req.QRType.String(), req.Amount)
}

// Helper methods with proper implementation...
func (s *service) validateRequest(req domainQR.GenerateQRRequest) error {
	return validation.ValidateQRRequest(req)
}

func (s *service) validatePayment(qr *models.QRCode, amount float64) error {
	if err := validation.ValidateQRPayment(qr, amount); err != nil {
		return err
	}
	return validation.CheckQRLimits(context.Background(), qr, amount)
}

func (s *service) processTransaction(ctx context.Context, tx *models.Transaction) error {
	return s.db.Transaction(func(dtx *gorm.DB) error {
		if err := s.walletSvc.Debit(ctx, tx.SenderID, tx.Amount); err != nil {
			return fmt.Errorf("failed to debit sender: %w", err)
		}

		if err := s.walletSvc.Credit(ctx, tx.ReceiverID, tx.Amount); err != nil {
			// Rollback the debit
			_ = s.walletSvc.Credit(ctx, tx.SenderID, tx.Amount)
			return fmt.Errorf("failed to credit receiver: %w", err)
		}

		tx.Status = "completed"
		if err := dtx.Create(tx).Error; err != nil {
			// Rollback the transfer
			_ = s.walletSvc.Credit(ctx, tx.SenderID, tx.Amount)
			_ = s.walletSvc.Debit(ctx, tx.ReceiverID, tx.Amount)
			return fmt.Errorf("failed to save transaction: %w", err)
		}

		return nil
	})
}

func (s *service) generateQRCode(ctx context.Context, req domainQR.GenerateQRRequest) (*models.QRCode, error) {
	if err := s.validateRequest(req); err != nil {
		log.Printf("QR validation failed: %v", err)
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	code, err := utils.GenerateSecureCode()
	if err != nil {
		log.Printf("Failed to generate secure code: %v", err)
		return nil, fmt.Errorf("failed to generate code: %w", err)
	}

	// Get default limits based on user type
	limits := DefaultLimits[req.UserType]

	qr := &models.QRCode{
		Code:         code,
		UserID:       req.UserID,
		UserType:     string(req.UserType),
		Type:         string(req.QRType),
		Amount:       req.Amount,
		ExpiresAt:    req.ExpiresAt,
		MaxUses:      req.MaxUses,
		Status:       "active",
		DailyLimit:   &limits.DailyLimit,
		MonthlyLimit: &limits.MonthlyLimit,
	}

	if err := s.db.WithContext(ctx).Create(qr).Error; err != nil {
		log.Printf("Failed to save QR code: %v", err)
		return nil, fmt.Errorf("failed to save QR code: %w", err)
	}

	return qr, nil
}

func (s *service) validateAndLockQR(ctx context.Context, code string) (*models.QRCode, error) {
	var qr models.QRCode
	if err := s.db.WithContext(ctx).Set("gorm:for_update", true).
		Where("code = ?", code).
		First(&qr).Error; err != nil {
		return nil, fmt.Errorf("QR code not found: %w", err)
	}

	if qr.Status != "active" {
		return nil, domainErrors.ErrQRInactive
	}

	if qr.ExpiresAt != nil && time.Now().After(*qr.ExpiresAt) {
		return nil, domainErrors.ErrQRExpired
	}

	if qr.MaxUses > 0 && qr.UsageCount >= qr.MaxUses {
		return nil, domainErrors.ErrQRLimitExceeded
	}

	return &qr, nil
}

func (s *service) updateQRUsage(ctx context.Context, qr *models.QRCode) error {
	qr.UsageCount++
	if qr.MaxUses > 0 && qr.UsageCount >= qr.MaxUses {
		qr.Status = "expired"
	}
	return s.db.WithContext(ctx).Save(qr).Error
}

func (s *service) GetUserReceiveQR(ctx context.Context, userID uint) (*models.QRCode, error) {
	var qr models.QRCode
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND type = ? AND user_type = ?", userID, string(domainQR.TypeReceive), "user").
		First(&qr).Error

	if err == gorm.ErrRecordNotFound {
		// Silently create new QR - this is expected for new users
		req := domainQR.GenerateQRRequest{
			UserID:    userID,
			UserType:  domainQR.UserTypeRegular,
			QRType:    domainQR.TypeReceive,
			MaxUses:   -1,
			ExpiresAt: nil,
		}
		return s.generateQRCode(ctx, req)
	}

	if err != nil {
		log.Printf("Error getting receive QR: %v", err) // Only log unexpected errors
		return nil, fmt.Errorf("failed to query receive QR: %w", err)
	}
	return &qr, nil
}

func (s *service) GetUserPaymentCodeQR(ctx context.Context, userID uint) (*models.QRCode, error) {
	var qr models.QRCode
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND type = ? AND user_type = ?", userID, string(domainQR.TypePaymentCode), "user").
		First(&qr).Error

	if err == gorm.ErrRecordNotFound {
		req := domainQR.GenerateQRRequest{
			UserID:    userID,
			UserType:  domainQR.UserTypeRegular,
			QRType:    domainQR.TypePaymentCode,
			MaxUses:   -1,
			ExpiresAt: nil,
		}
		return s.generateQRCode(ctx, req)
	}
	return &qr, err
}
