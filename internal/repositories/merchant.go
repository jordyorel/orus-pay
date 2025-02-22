package repositories

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"orus/internal/models"
)

func GetMerchantByUserID(userID uint) (*models.Merchant, error) {
	var merchant models.Merchant
	if err := DB.Where("user_id = ?", userID).First(&merchant).Error; err != nil {
		return nil, err
	}
	return &merchant, nil
}

func CreateMerchant(merchant *models.Merchant) error {
	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Clean up any existing records
	if err := tx.Unscoped().Where("user_id = ?", merchant.UserID).Delete(&models.Merchant{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Reset the sequence if needed
	if err := tx.Exec("ALTER SEQUENCE merchants_id_seq RESTART WITH 1").Error; err != nil {
		tx.Rollback()
		return err
	}

	// Create the new merchant (let DB handle ID auto-increment)
	if err := tx.Create(merchant).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Update user's status
	if err := tx.Model(&models.User{}).
		Where("id = ?", merchant.UserID).
		Update("merchant_profile_status", "completed").Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func UpdateMerchant(merchant *models.Merchant) error {
	return DB.Save(merchant).Error
}

func GetMerchantTransactions(merchantID uint) ([]models.Transaction, error) {
	var transactions []models.Transaction
	err := DB.Where("merchant_id = ?", merchantID).
		Order("created_at DESC").
		Find(&transactions).Error
	return transactions, err
}

func GenerateMerchantAPIKey(merchantID uint) (string, error) {
	// Generate random bytes for API key
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	apiKey := hex.EncodeToString(bytes)

	// Update merchant with new API key
	result := DB.Model(&models.Merchant{}).
		Where("user_id = ?", merchantID).
		Update("api_key", apiKey)
	if result.Error != nil {
		return "", result.Error
	}
	if result.RowsAffected == 0 {
		return "", fmt.Errorf("merchant not found")
	}

	return apiKey, nil
}

func SetMerchantWebhookURL(merchantID uint, webhookURL string) error {
	result := DB.Model(&models.Merchant{}).
		Where("user_id = ?", merchantID).
		Update("webhook_url", webhookURL)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("merchant not found")
	}
	return nil
}
