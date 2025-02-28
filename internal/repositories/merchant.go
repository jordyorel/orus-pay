package repositories

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"orus/internal/models"

	"gorm.io/gorm"
)

type MerchantRepository interface {
	GetByID(id uint) (*models.Merchant, error)
	GetByUserID(userID uint) (*models.Merchant, error)
	Create(merchant *models.Merchant) error
	Update(merchant *models.Merchant) error
}

type merchantRepository struct {
	db *gorm.DB
}

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
	// Ensure we're updating an existing record by using the ID
	if merchant.ID == 0 {
		return errors.New("cannot update merchant with ID 0")
	}
	return DB.Model(&models.Merchant{}).Where("id = ?", merchant.ID).Updates(merchant).Error
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

func GetMerchantStaticQR(userID uint) (*models.QRCode, error) {
	var qr models.QRCode
	err := DB.Where("user_id = ? AND user_type = ? AND type = ?",
		userID, "merchant", models.QRTypeStatic).First(&qr).Error
	if err != nil {
		return nil, err
	}
	return &qr, nil
}

func NewMerchantRepository(db *gorm.DB) MerchantRepository {
	return &merchantRepository{
		db: db,
	}
}

func (r *merchantRepository) GetByID(id uint) (*models.Merchant, error) {
	var merchant models.Merchant
	err := r.db.First(&merchant, id).Error
	return &merchant, err
}

func (r *merchantRepository) GetByUserID(userID uint) (*models.Merchant, error) {
	var merchant models.Merchant
	err := r.db.Where("user_id = ?", userID).First(&merchant).Error
	return &merchant, err
}

func (r *merchantRepository) Create(merchant *models.Merchant) error {
	return r.db.Create(merchant).Error
}

func (r *merchantRepository) Update(merchant *models.Merchant) error {
	return r.db.Save(merchant).Error
}
