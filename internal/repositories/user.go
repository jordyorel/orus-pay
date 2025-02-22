package repositories

import (
	"fmt"
	"log"
	"orus/internal/models"
	"orus/internal/utils"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	userCacheExpiration = 24 * time.Hour
)

func getUserCacheKeyByID(id uint) string {
	return fmt.Sprintf("user:id:%d", id)
}

func GetUserCacheKeyByEmail(email string) string {
	return fmt.Sprintf("user:email:%s", email)
}

func GetUserCacheKeyByPhone(phone string) string {
	return fmt.Sprintf("user:phone:%s", phone)
}

func GetUserByEmail(email string) (*models.User, error) {
	// Try cache first
	cacheKey := GetUserCacheKeyByEmail(email)
	cachedUser, err := cacheGetUser(cacheKey)
	if err == nil {
		log.Printf("Cache hit for user email: %s", email)
		return cachedUser, nil
	}
	if err != redis.Nil {
		log.Printf("Cache error for email %s: %v", email, err)
	}

	// Cache miss, query database
	var user models.User
	if err := DB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}

	// Update cache async
	go func() {
		if err := cacheSetUser(cacheKey, &user, userCacheExpiration); err != nil {
			log.Printf("Failed to cache user %s: %v", email, err)
		}
	}()

	return &user, nil
}

func GetUserByID(userID uint) (*models.User, error) {
	cacheKey := getUserCacheKeyByID(userID)
	cachedUser, err := cacheGetUser(cacheKey)
	if err == nil {
		log.Printf("Cache hit for user ID: %d", userID)
		return cachedUser, nil
	}
	if err != redis.Nil {
		log.Printf("Cache error for ID %d: %v", userID, err)
	}

	var user models.User
	// err = DB.First(&user, userID).Error
	err = DB.Where("id = ?", userID).First(&user).Error
	if err != nil {
		return nil, err
	}

	go func() {
		if err := cacheSetUser(cacheKey, &user, userCacheExpiration); err != nil {
			log.Printf("Failed to cache user by ID: %v", err)
		}
	}()

	return &user, nil
}

func GetUserByPhone(phone string) (*models.User, error) {
	cacheKey := GetUserCacheKeyByPhone(phone)
	cachedUser, err := cacheGetUser(cacheKey)
	if err == nil {
		log.Printf("Cache hit for user phone: %s", phone)
		return cachedUser, nil
	}
	if err != redis.Nil {
		log.Printf("Cache error for phone %s: %v", phone, err)
	}

	var user models.User
	result := DB.Where("phone = ?", phone).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}

	go func() {
		if err := cacheSetUser(cacheKey, &user, userCacheExpiration); err != nil {
			log.Printf("Failed to cache user by phone: %v", err)
		}
	}()

	return &user, nil
}

func CreateUser(user *models.User) (*models.User, *models.QRCode, error) {
	// Standardize role/type mapping
	if user.Role == "user" {
		user.Role = "regular"
	}
	user.UserType = user.Role // Make UserType match Role for consistency

	var qrCode *models.QRCode

	err := DB.Transaction(func(tx *gorm.DB) error {
		// Create user
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		// Create wallet with proper linking
		wallet := &models.Wallet{
			UserID:   user.ID,
			Balance:  0,
			Currency: "USD",
		}
		if err := tx.Create(wallet).Error; err != nil {
			return err
		}

		// Update user with wallet ID
		user.WalletID = &wallet.ID
		if err := tx.Save(user).Error; err != nil {
			return err
		}

		// Create merchant profile if needed
		if user.Role == "merchant" {
			merchant := &models.Merchant{
				UserID:          user.ID,
				BusinessName:    fmt.Sprintf("Business-%d", user.ID),
				BusinessType:    "unspecified",
				BusinessAddress: "pending",
				IsActive:        true,
			}
			if err := tx.Create(merchant).Error; err != nil {
				return err
			}
		}

		// Generate static QR code with proper limits
		code, err := utils.GenerateSecureCode()
		if err != nil {
			return err
		}

		qrCode = &models.QRCode{
			Code:     code,
			UserID:   user.ID,
			UserType: user.Role, // Use standardized role
			Type:     "static",
			Status:   "active",
			MaxUses:  -1,
		}

		// Set limits based on role
		if user.Role == "merchant" {
			dailyLimit := float64(10000)
			monthlyLimit := float64(100000)
			qrCode.DailyLimit = &dailyLimit
			qrCode.MonthlyLimit = &monthlyLimit
		} else {
			dailyLimit := float64(1000)
			monthlyLimit := float64(5000)
			qrCode.DailyLimit = &dailyLimit
			qrCode.MonthlyLimit = &monthlyLimit
		}

		return tx.Create(qrCode).Error
	})

	if err != nil {
		return nil, nil, err
	}

	// Invalidate any existing cache entries
	cacheKeyEmail := GetUserCacheKeyByEmail(user.Email)
	cacheKeyPhone := GetUserCacheKeyByPhone(user.Phone)
	cacheKeyID := getUserCacheKeyByID(user.ID)
	RedisClient.Del(RedisCtx, cacheKeyEmail, cacheKeyPhone, cacheKeyID)

	// Fetch fresh user data from DB
	freshUser, err := GetUserByID(user.ID)
	if err != nil {
		return nil, nil, err
	}

	return freshUser, qrCode, nil
}

// internal/repositories/user.go
func GetUsersPaginated(limit, offset int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	// Get total count
	if err := DB.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := DB.Limit(limit).Offset(offset).Find(&users).Error
	return users, total, err
}

func UpdateUser(user *models.User) error {
	result := DB.Save(user)
	return result.Error
}

func InvalidateUserCache(userID uint) {
	keys := []string{
		getUserCacheKeyByID(userID),
		// Add other cache keys if needed
	}
	RedisClient.Del(RedisCtx, keys...)
}

func DeleteUserByID(userID string) error {
	// Convert string ID to uint
	id, err := strconv.ParseUint(userID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid user ID format")
	}
	uintID := uint(id)

	// First invalidate cache
	InvalidateUserCache(uintID)

	// Then delete from database
	result := DB.Delete(&models.User{}, uintID)
	if result.Error != nil {
		return result.Error
	}

	// Check if any row was affected
	if result.RowsAffected == 0 {
		return fmt.Errorf("no user found with ID %s", userID)
	}

	return nil
}

func IncrementUserTokenVersion(userID uint) error {
	// First, fetch the user to get the email and phone values.
	var user models.User
	if err := DB.First(&user, userID).Error; err != nil {
		return err
	}

	// Invalidate all cache keys for the user
	cacheKeyID := getUserCacheKeyByID(userID)
	cacheKeyEmail := GetUserCacheKeyByEmail(user.Email)
	cacheKeyPhone := GetUserCacheKeyByPhone(user.Phone)
	RedisClient.Del(RedisCtx, cacheKeyID, cacheKeyEmail, cacheKeyPhone)

	// Increment the token version and save to the database
	user.TokenVersion++
	return DB.Save(&user).Error
}
