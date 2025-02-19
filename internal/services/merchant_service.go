package services

import (
	"errors"
	"fmt"
	"orus/internal/models"
	"orus/internal/repositories"

	"gorm.io/gorm"
)

type MerchantService struct {
	feeCalculator *FeeCalculator
}

func NewMerchantService() *MerchantService {
	return &MerchantService{
		feeCalculator: NewFeeCalculator(),
	}
}

func (s *MerchantService) CreateMerchant(merchant *models.Merchant) error {
	// Verify user exists and has merchant role
	user, err := repositories.GetUserByID(merchant.UserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if user.Role != "merchant" {
		return fmt.Errorf("user must have merchant role to create merchant profile")
	}

	// Check if merchant profile already exists
	existingMerchant, err := repositories.GetMerchantByUserID(merchant.UserID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("error checking existing merchant: %w", err)
	}

	// If merchant exists, only update the additional fields
	if existingMerchant != nil {
		// Keep existing basic info, update only the new fields
		updates := map[string]interface{}{
			"business_id":         merchant.BusinessID,
			"tax_id":              merchant.TaxID,
			"website":             merchant.Website,
			"merchant_category":   merchant.MerchantCategory,
			"legal_entity_type":   merchant.LegalEntityType,
			"registration_number": merchant.RegistrationNumber,
			"year_established":    merchant.YearEstablished,
			"support_email":       merchant.SupportEmail,
			"support_phone":       merchant.SupportPhone,
			"verification_status": "pending_review",
		}

		return repositories.DB.Model(existingMerchant).Updates(updates).Error
	}

	// Set default values for new merchant
	merchant.VerificationStatus = "pending_review"
	merchant.ProcessingFeeRate = 2.5

	return repositories.DB.Transaction(func(tx *gorm.DB) error {
		// Create merchant profile
		if err := tx.Create(merchant).Error; err != nil {
			return fmt.Errorf("failed to create merchant: %w", err)
		}

		// Create default merchant limits
		limits := &models.MerchantLimits{
			MerchantID:              merchant.ID,
			DailyTransactionLimit:   10000,
			MonthlyTransactionLimit: 100000,
			SingleTransactionLimit:  5000,
			MinTransactionAmount:    1,
			MaxTransactionAmount:    5000,
			ConcurrentTransactions:  10,
			AllowedCurrencies:       []string{"USD", "EUR"},
		}

		if err := tx.Create(limits).Error; err != nil {
			return fmt.Errorf("failed to create merchant limits: %w", err)
		}

		// Update user status to indicate merchant profile needs completion
		if err := tx.Model(&models.User{}).Where("id = ?", merchant.UserID).
			Update("merchant_profile_status", "pending_completion").Error; err != nil {
			return fmt.Errorf("failed to update user status: %w", err)
		}

		return nil
	})
}

func (s *MerchantService) UpdateLimits(merchantID uint, limits models.MerchantLimits) error {
	return repositories.DB.Model(&models.Merchant{}).Where("id = ?", merchantID).
		Update("limits", limits).Error
}

func (s *MerchantService) ProcessTransaction(merchantID uint, amount float64) error {
	var merchant models.Merchant
	if err := repositories.DB.Preload("Limits").First(&merchant, merchantID).Error; err != nil {
		return fmt.Errorf("merchant not found: %w", err)
	}

	// Check limits
	if amount > merchant.Limits.SingleTransactionLimit {
		return fmt.Errorf("amount exceeds single transaction limit")
	}

	// Calculate fee using the injected calculator
	fee := s.feeCalculator.CalculateTransactionFee(amount, models.UserTypeMerchant)

	// Update merchant stats
	merchant.TotalTransactions++
	merchant.TotalVolume += amount
	merchant.MonthlyVolume += amount
	merchant.ProcessingFeeRate = fee / amount * 100 // Convert to percentage

	return repositories.DB.Save(&merchant).Error
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
