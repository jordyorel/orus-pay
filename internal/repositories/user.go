package repositories

import (
	"fmt"
	"log"
	"orus/internal/models"
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

func CreateUser(user *models.User) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		// Check for existing user
		var existingUser models.User
		if err := tx.Where("email = ? OR phone = ?", user.Email, user.Phone).First(&existingUser).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				return fmt.Errorf("error checking existing user: %w", err)
			}
		} else {
			return fmt.Errorf("user with this email or phone already exists")
		}

		// Create user first with null wallet_id
		user.WalletID = nil
		if err := tx.Create(user).Error; err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		// Create wallet
		wallet := &models.Wallet{
			UserID:   user.ID,
			Balance:  0,
			Currency: "USD",
			QRCodeID: fmt.Sprintf("orus://pay?user_id=%d", user.ID),
		}
		if err := tx.Create(wallet).Error; err != nil {
			return fmt.Errorf("failed to create wallet: %w", err)
		}

		// Update user's wallet_id
		walletIDUint := uint(wallet.ID)
		user.WalletID = &walletIDUint
		if err := tx.Model(user).Update("wallet_id", wallet.ID).Error; err != nil {
			return fmt.Errorf("failed to update user with wallet: %w", err)
		}

		// Invalidate all possible cache keys for this user
		go func() {
			RedisClient.Del(RedisCtx,
				getUserCacheKeyByID(user.ID),
				GetUserCacheKeyByEmail(user.Email),
				GetUserCacheKeyByPhone(user.Phone),
			)
		}()

		return nil
	})
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
