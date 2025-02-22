package services

import (
	"errors"
	"fmt"
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
}

func NewMerchantService() *MerchantService {
	return &MerchantService{
		feeCalculator:      NewFeeCalculator(),
		transactionService: NewTransactionService(),
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
	var result *models.Transaction
	err := repositories.DB.Transaction(func(dbTx *gorm.DB) error {
		// Get merchant by user_id instead of id
		var merchant models.Merchant
		if err := dbTx.Where("user_id = ?", *tx.MerchantID).First(&merchant).Error; err != nil {
			return fmt.Errorf("merchant not found: %w", err)
		}

		if !merchant.IsActive {
			return ErrMerchantInactive
		}

		// Process the transaction
		processedTx, err := s.transactionService.ProcessTransaction(tx)
		if err != nil {
			return err
		}
		result = processedTx

		// Update merchant statistics
		merchant.TotalTransactions++
		merchant.TotalVolume += tx.Amount
		return dbTx.Save(&merchant).Error
	})

	return result, err
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
