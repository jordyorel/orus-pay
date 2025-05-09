package merchant

import (
	"context"
	"errors"
	"fmt"
	"log"
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/services"
	"orus/internal/services/qr_code"
	"orus/internal/services/transaction"
	"orus/internal/services/wallet"

	"gorm.io/gorm"
)

type Service struct {
	qrService          qr_code.Service
	transactionService transaction.Service
	walletService      wallet.Service
	feeCalculator      *services.FeeCalculator
}

func NewService(
	qrSvc qr_code.Service,
	txSvc transaction.Service,
	walletSvc wallet.Service,
) *Service {
	return &Service{
		qrService:          qrSvc,
		transactionService: txSvc,
		walletService:      walletSvc,
		feeCalculator:      services.NewFeeCalculator(),
	}
}

func (s *Service) CreateMerchant(merchant *models.Merchant) (*models.Merchant, error) {
	log.Printf("Creating new merchant for user ID: %d", merchant.UserID)

	// Check for existing merchant
	existingMerchant, err := repositories.GetMerchantByUserID(merchant.UserID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existingMerchant != nil {
		return nil, errors.New("merchant profile already exists")
	}

	// Set defaults
	merchant.RiskScore = int(calculateInitialRiskScore(merchant))
	merchant.ComplianceLevel = determineComplianceLevel(float64(merchant.RiskScore))
	merchant.DailyTransactionLimit = DefaultDailyLimit
	merchant.MonthlyTransactionLimit = DefaultMonthlyLimit
	merchant.MinTransactionAmount = DefaultMinAmount
	merchant.MaxTransactionAmount = DefaultMaxAmount
	merchant.Status = "active"

	// Create merchant profile without QR codes
	if err := repositories.DB.Create(merchant).Error; err != nil {
		return nil, err
	}
	return merchant, nil
}

func (s *Service) ProcessDirectCharge(merchantID uint, input ChargeInput) (*models.Transaction, error) {
	// Validate the payment code
	var qrCode models.QRCode
	if err := repositories.DB.Where("code = ? AND status = ?", input.PaymentCode, "active").First(&qrCode).Error; err != nil {
		return nil, fmt.Errorf("invalid payment code: %w", err)
	}

	// Verify this is a payment code QR, not a receive QR
	if qrCode.Type != string(qr_code.TypePaymentCode) {
		return nil, fmt.Errorf("invalid QR type: merchant can only scan payment codes")
	}

	// Get merchant details
	merchant, err := repositories.GetMerchantByUserID(merchantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create default merchant profile
			merchant = &models.Merchant{
				UserID:                  merchantID,
				BusinessName:            "Default Business",
				BusinessType:            "retail",
				Status:                  "active",
				ComplianceLevel:         "medium_risk",
				RiskScore:               50,
				DailyTransactionLimit:   DefaultDailyLimit,
				MonthlyTransactionLimit: DefaultMonthlyLimit,
				MinTransactionAmount:    DefaultMinAmount,
				MaxTransactionAmount:    DefaultMaxAmount,
			}

			if err := repositories.DB.Create(merchant).Error; err != nil {
				return nil, fmt.Errorf("failed to create merchant profile: %w", err)
			}
		} else {
			return nil, fmt.Errorf("merchant not found: %w", err)
		}
	}

	// Get customer ID from QR code
	customerID := qrCode.UserID

	// Verify customer wallet exists and has sufficient balance
	if err := s.walletService.ValidateBalance(context.Background(), customerID, input.Amount); err != nil {
		return nil, fmt.Errorf("insufficient balance: %w", err)
	}

	metadata := map[string]interface{}{
		"scanner_role":      "merchant",
		"payment_code":      input.PaymentCode,
		"merchant_id":       merchantID,
		"merchant_name":     merchant.BusinessName,
		"merchant_category": merchant.BusinessType,
		"payment_type":      "merchant_scan",
		"device_type":       "pos",
	}

	tx, err := s.qrService.ProcessQRPayment(
		context.Background(),
		input.PaymentCode,
		input.Amount,
		merchantID,
		input.Description,
		metadata,
	)
	if err != nil {
		return nil, err
	}

	// Enrich transaction with merchant details
	tx.MerchantID = &merchant.ID
	tx.MerchantName = merchant.BusinessName
	tx.MerchantCategory = merchant.BusinessType

	// Update the transaction record
	if err := repositories.DB.Save(tx).Error; err != nil {
		return nil, fmt.Errorf("failed to update transaction with merchant details: %w", err)
	}

	return tx, nil
}

// Move all merchant service methods here

func calculateInitialRiskScore(merchant *models.Merchant) float64 {
	var score float64 = 50 // Base score

	if merchant.BusinessType == "high_risk" {
		score += 20
	}

	if merchant.MonthlyVolume > 50000 {
		score += 10
	}

	return score
}

func determineComplianceLevel(riskScore float64) string {
	switch {
	case riskScore < 30:
		return "low_risk"
	case riskScore < 70:
		return "medium_risk"
	default:
		return "high_risk"
	}
}

func (s *Service) processTransaction(tx *models.Transaction) (*models.Transaction, error) {
	ctx := context.Background()

	merchant, err := repositories.GetMerchantByUserID(tx.ReceiverID)
	if err != nil {
		return nil, err
	}

	// Enrich transaction with merchant details
	tx.MerchantID = &merchant.ID
	tx.MerchantName = merchant.BusinessName
	tx.MerchantCategory = merchant.BusinessType
	tx.PaymentMethod = "WALLET"

	// Calculate fee
	fee := s.feeCalculator.CalculateFee(tx.Amount)
	tx.Fee = fee

	// Process the actual transaction
	err = repositories.DB.Transaction(func(db *gorm.DB) error {
		if err := s.walletService.Debit(ctx, tx.SenderID, tx.Amount+fee); err != nil {
			return err
		}

		if err := s.walletService.Credit(ctx, tx.ReceiverID, tx.Amount); err != nil {
			_ = s.walletService.Credit(ctx, tx.SenderID, tx.Amount+fee)
			return err
		}

		tx.Status = "completed"
		return db.Save(tx).Error
	})

	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (s *Service) GetMerchant(ctx context.Context, userID uint) (*models.Merchant, error) {
	return repositories.GetMerchantByUserID(userID)
}

func (s *Service) UpdateMerchantProfile(merchantID uint, input UpdateMerchantInput) error {
	merchant, err := repositories.GetMerchantByUserID(merchantID)
	if err != nil {
		return err
	}

	merchant.BusinessName = input.BusinessName
	merchant.BusinessType = input.BusinessType
	merchant.BusinessAddress = input.BusinessAddress
	merchant.ProcessingFeeRate = input.ProcessingFee
	merchant.WebhookURL = input.WebhookURL

	return repositories.UpdateMerchant(merchant)
}

func (s *Service) ProcessQRPayment(ctx context.Context, merchantID uint, input QRPaymentInput) (*models.Transaction, error) {
	tx := &models.Transaction{
		Type:        models.TransactionTypeQRPayment,
		ReceiverID:  merchantID,
		Amount:      input.Amount,
		Description: input.Description,
		Status:      "pending",
		Currency:    "USD",
	}

	return s.processTransaction(tx)
}

func (s *Service) GenerateAPIKey(merchantID uint) (string, error) {
	return repositories.GenerateMerchantAPIKey(merchantID)
}

func (s *Service) SetWebhookURL(merchantID uint, webhookURL string) error {
	return repositories.SetMerchantWebhookURL(merchantID, webhookURL)
}
