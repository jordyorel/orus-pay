package repositories

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"orus/internal/models"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	walletCacheExpiration = 24 * time.Hour
)

func getWalletCacheKeyByUserID(userID uint) string {
	return fmt.Sprintf("wallet:user:%d", userID)
}

func getWalletCacheKeyByQRCodeID(qrCodeID string) string {
	return fmt.Sprintf("wallet:qrcode:%s", qrCodeID)
}

func cacheGetWallet(key string) (*models.Wallet, error) {
	val, err := RedisClient.Get(RedisCtx, key).Result()
	if err != nil {
		return nil, err
	}

	var wallet models.Wallet
	if err := json.Unmarshal([]byte(val), &wallet); err != nil {
		return nil, err
	}
	return &wallet, nil
}

func cacheSetWallet(key string, wallet *models.Wallet, expiration time.Duration) error {
	walletBytes, err := json.Marshal(wallet)
	if err != nil {
		return err
	}
	return RedisClient.Set(RedisCtx, key, walletBytes, expiration).Err()
}

func GetWalletByUserID(userID uint) (*models.Wallet, error) {
	cacheKey := getWalletCacheKeyByUserID(userID)
	cachedWallet, err := cacheGetWallet(cacheKey)
	if err == nil {
		log.Printf("Cache hit for wallet user ID: %d", userID)
		return cachedWallet, nil
	}
	if err != redis.Nil {
		log.Printf("Cache error for user ID %d: %v", userID, err)
	}

	var wallet models.Wallet
	err = DB.Where("user_id = ?", userID).First(&wallet).Error
	if err != nil {
		return nil, err
	}

	go func() {
		if err := cacheSetWallet(cacheKey, &wallet, walletCacheExpiration); err != nil {
			log.Printf("Failed to cache wallet for user %d: %v", userID, err)
		}
	}()

	return &wallet, nil
}

func GetWalletByQRCodeID(qrCodeID string) (*models.Wallet, error) {
	cacheKeyQR := getWalletCacheKeyByQRCodeID(qrCodeID)
	cachedWallet, err := cacheGetWallet(cacheKeyQR)
	if err == nil {
		log.Printf("Cache hit for wallet QR code ID: %s", qrCodeID)
		return cachedWallet, nil
	}
	if err != redis.Nil {
		log.Printf("Cache error for QR code ID %s: %v", qrCodeID, err)
	}

	u, err := url.Parse(qrCodeID)
	if err != nil {
		log.Printf("Error parsing QR code: %v", err)
		return nil, fmt.Errorf("invalid QR code format")
	}

	userIDStr := u.Query().Get("user_id")
	if userIDStr == "" {
		log.Println("Invalid QR code: missing user_id")
		return nil, fmt.Errorf("invalid QR code: missing user_id")
	}

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		log.Printf("Error converting user_id to int: %v", err)
		return nil, fmt.Errorf("invalid user_id in QR code")
	}

	wallet, err := GetWalletByUserID(uint(userID))
	if err != nil {
		return nil, err
	}

	go func() {
		if err := cacheSetWallet(cacheKeyQR, wallet, walletCacheExpiration); err != nil {
			log.Printf("Failed to cache wallet by QR code ID %s: %v", qrCodeID, err)
		}
	}()

	return wallet, nil
}

func CreateWallet(wallet *models.Wallet) error {
	if wallet.QRCodeID == "" {
		wallet.QRCodeID = generateQRCode(wallet.UserID)
	}
	err := DB.Create(wallet).Error
	if err != nil {
		return err
	}

	cacheKeyUser := getWalletCacheKeyByUserID(wallet.UserID)
	cacheKeyQR := getWalletCacheKeyByQRCodeID(wallet.QRCodeID)

	go func() {
		if err := cacheSetWallet(cacheKeyUser, wallet, walletCacheExpiration); err != nil {
			log.Printf("Failed to cache wallet for user %d: %v", wallet.UserID, err)
		}
		if err := cacheSetWallet(cacheKeyQR, wallet, walletCacheExpiration); err != nil {
			log.Printf("Failed to cache wallet by QR code %s: %v", wallet.QRCodeID, err)
		}
	}()

	return nil
}

func UpdateWallet(wallet *models.Wallet) error {
	// Only update balance and updated_at fields
	result := DB.Model(wallet).Updates(map[string]interface{}{
		"balance":    wallet.Balance,
		"updated_at": time.Now(),
	})

	if result.Error != nil {
		log.Printf("Error updating wallet: %v", result.Error)
		return result.Error
	}

	// Invalidate wallet cache
	cacheKey := getWalletCacheKeyByUserID(wallet.UserID)
	RedisClient.Del(RedisCtx, cacheKey)

	return nil
}

func TopUpWallet(userID uint, amount float64) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var wallet models.Wallet
		if err := tx.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
			return err
		}

		wallet.Balance += amount

		// Only update balance
		if err := tx.Model(&wallet).Update("balance", wallet.Balance).Error; err != nil {
			return err
		}

		// Invalidate cache
		cacheKey := getWalletCacheKeyByUserID(userID)
		RedisClient.Del(RedisCtx, cacheKey)

		return nil
	})
}

// generateQRCode generates a fixed QR code ID for the wallet
func generateQRCode(userID uint) string {
	return "orus://pay?user_id=" + fmt.Sprintf("%d", userID)
}

func validateCardInput(card models.CreateCreditCard) error {
	if card.CardNumber == "" {
		return errors.New("card number is required")
	}
	if card.ExpiryMonth == "" || card.ExpiryYear == "" {
		return errors.New("expiry date is required")
	}

	month, err := strconv.Atoi(card.ExpiryMonth)
	if err != nil || month < 1 || month > 12 {
		return errors.New("invalid expiry month")
	}

	year, err := strconv.Atoi(card.ExpiryYear)
	if err != nil {
		return errors.New("invalid expiry year")
	}

	now := time.Now()
	if year < now.Year() || (year == now.Year() && month < int(now.Month())) {
		return errors.New("card has expired")
	}

	return nil
}

func GetWalletsPaginated(limit, offset int) ([]models.Wallet, int64, error) {
	var wallets []models.Wallet
	var total int64

	// Fetch wallets with pagination
	result := DB.Model(&models.Wallet{}).Count(&total).Limit(limit).Offset(offset).Find(&wallets)
	if result.Error != nil {
		return nil, 0, result.Error
	}

	return wallets, total, nil
}
