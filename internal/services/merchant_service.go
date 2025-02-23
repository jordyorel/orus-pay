package services

import (
	"errors"
	"log"
	"orus/internal/models"
	"orus/internal/repositories"

	"gorm.io/gorm"
)

var (
	ErrMerchantInactive = errors.New("merchant is not active")
)

type MerchantService struct {
	feeCalculator      *FeeCalculator
	transactionService *TransactionService
	walletService      *WalletService
}

func NewMerchantService() *MerchantService {
	return &MerchantService{
		feeCalculator:      NewFeeCalculator(),
		transactionService: NewTransactionService(),
		walletService:      NewWalletService(),
	}
}

func (s *MerchantService) CreateMerchant(merchant *models.Merchant) error {
	log.Printf("Checking for existing merchant for user ID: %d", merchant.UserID)

	existingMerchant, err := repositories.GetMerchantByUserID(merchant.UserID)
	if err != nil {
		log.Printf("Error checking existing merchant: %v", err)
		return err
	}

	if existingMerchant != nil {
		log.Printf("Found existing merchant for user ID: %d", merchant.UserID)
		return errors.New("merchant profile already exists for this user")
	}

	log.Printf("Creating new merchant for user ID: %d", merchant.UserID)
	// Set initial risk score and compliance level
	merchant.RiskScore = int(calculateInitialRiskScore(merchant))
	merchant.ComplianceLevel = determineComplianceLevel(float64(merchant.RiskScore))

	// Create new merchant
	return repositories.CreateMerchant(merchant)
}

func (s *MerchantService) UpdateLimits(merchantID uint, limits models.MerchantLimits) error {
	return repositories.DB.Model(&models.Merchant{}).Where("id = ?", merchantID).
		Update("limits", limits).Error
}

func (s *MerchantService) ProcessTransaction(tx *models.Transaction) (*models.Transaction, error) {
	// Get merchant details
	merchant, err := repositories.GetMerchantByUserID(tx.ReceiverID)
	if err != nil {
		return nil, err
	}

	// Enrich transaction with merchant details
	tx.MerchantID = &merchant.ID
	tx.MerchantName = merchant.BusinessName
	tx.MerchantCategory = merchant.BusinessType
	tx.PaymentMethod = "WALLET"

	if tx.Type == models.TransactionTypeMerchantScan {
		// When merchant scans customer QR
		tx.PaymentType = "QR_SCAN"
		// Get customer's payment QR
		qr, err := repositories.GetUserPaymentQR(tx.SenderID)
		if err == nil {
			tx.QRCodeID = qr.Code
			tx.QRType = qr.Type
			tx.QROwnerID = qr.UserID
			tx.QROwnerType = "user"
		}
	} else if tx.Type == models.TransactionTypeQRPayment {
		// When customer scans merchant QR
		tx.PaymentType = "QR_PAYMENT"
		// QR details should already be set
	}

	// Calculate fee
	fee := s.feeCalculator.CalculateFee(tx.Amount)
	tx.Fee = fee

	// Process the actual transaction
	err = repositories.DB.Transaction(func(db *gorm.DB) error {
		// Debit sender
		if err := s.walletService.Debit(tx.SenderID, tx.Amount+fee); err != nil {
			return err
		}

		// Credit merchant
		if err := s.walletService.Credit(tx.ReceiverID, tx.Amount); err != nil {
			// Rollback sender's debit
			_ = s.walletService.Credit(tx.SenderID, tx.Amount+fee)
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

func calculateInitialRiskScore(merchant *models.Merchant) float64 {
	// Implement risk scoring logic
	var score float64 = 50 // Base score

	// Add scoring logic based on business type, volume, etc.
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

func (s *MerchantService) GetMerchant(userID uint) (*models.Merchant, error) {
	return repositories.GetMerchantByUserID(userID)
}

func (s *MerchantService) UpdateMerchant(merchant *models.Merchant) error {
	updates := map[string]interface{}{
		"business_name":    merchant.BusinessName,
		"business_type":    merchant.BusinessType,
		"business_address": merchant.BusinessAddress,
		"is_active":        merchant.IsActive,
	}

	return repositories.DB.Model(merchant).Updates(updates).Error
}
