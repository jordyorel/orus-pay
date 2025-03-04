package repositories

import (
	"context"
	"fmt"
	"log"
	"math"
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

func GetWalletByQRCodeID(qrCodeID string) (*models.Wallet, error) {
	cacheKeyQR := CacheService.GenerateKey("wallet", "qrcode", qrCodeID)
	var cachedWallet models.Wallet
	found, err := CacheService.Get(context.Background(), cacheKeyQR, &cachedWallet)
	if found && err == nil {
		log.Printf("Cache hit for wallet QR code ID: %s", qrCodeID)
		return &cachedWallet, nil
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

	// Cache the result
	if err := CacheService.SetWithTTL(context.Background(), cacheKeyQR, wallet, walletCacheExpiration); err != nil {
		log.Printf("Failed to cache wallet by QR code ID %s: %v", qrCodeID, err)
	}

	return wallet, nil
}

func CreateWallet(wallet *models.Wallet) error {
	err := DB.Create(wallet).Error
	if err != nil {
		return err
	}

	// Cache only by user ID now
	cacheKey := CacheService.GenerateKey("wallet", "user", wallet.UserID)
	if err := CacheService.SetWithTTL(context.Background(), cacheKey, wallet, walletCacheExpiration); err != nil {
		log.Printf("Failed to cache wallet for user %d: %v", wallet.UserID, err)
	}

	return nil
}

func TopUpWallet(userID uint, amount float64) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var wallet models.Wallet
		if err := tx.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
			return err
		}

		wallet.Balance += amount
		if err := tx.Save(&wallet).Error; err != nil {
			return err
		}

		// Invalidate cache
		cacheKey := CacheService.GenerateKey("wallet", "user", userID)
		CacheService.Delete(context.Background(), cacheKey)

		return nil
	})
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

func GetWalletByUserIDForUpdate(userID uint) (*models.Wallet, error) {
	var wallet models.Wallet
	if err := DB.Set("gorm:for_update", true).Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return nil, err
	}
	return &wallet, nil
}

func UpdateWalletBalance(tx *gorm.DB, walletID uint, newBalance float64) error {
	// Round to 2 decimal places
	newBalance = math.Round(newBalance*100) / 100

	return tx.Model(&models.Wallet{}).
		Where("id = ?", walletID).
		Update("balance", newBalance).Error
}

func ResetWalletBalance(userID uint) error {
	return DB.Model(&models.Wallet{}).
		Where("user_id = ?", userID).
		Update("balance", 0).Error
}

func GetWalletByUserID(userID uint) (*models.Wallet, error) {
	// Try cache first
	key := CacheService.GenerateKey("wallet", "user", userID)
	var wallet models.Wallet
	found, _ := CacheService.Get(context.Background(), key, &wallet)
	if found {
		return &wallet, nil
	}

	// Cache miss, fetch from database
	if err := DB.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		return nil, err
	}

	// Cache the result
	CacheService.SetWithTTL(context.Background(), key, wallet, 30*time.Minute)

	return &wallet, nil
}
