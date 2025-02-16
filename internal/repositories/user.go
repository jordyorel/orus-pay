package repositories

import (
	"fmt"
	"log"
	"orus/internal/models"
	"strconv"
	"strings"
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
	err = DB.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}

	// Update cache async to avoid blocking
	go func() {
		if err := cacheSetUser(cacheKey, &user, userCacheExpiration); err != nil {
			log.Printf("Failed to cache user by email: %v", err)
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
	if DB == nil {
		return fmt.Errorf("database connection is nil")
	}

	var existingUser models.User
	err := DB.Where("email = ? AND deleted_at IS NULL", user.Email).First(&existingUser).Error // Check with deleted_at!
	if err == nil || err != gorm.ErrRecordNotFound {
		return fmt.Errorf("user with email %s already exists", user.Email)
	}

	err = DB.Where("phone = ? AND deleted_at IS NULL", user.Phone).First(&existingUser).Error // Check with deleted_at!
	if err == nil || err != gorm.ErrRecordNotFound {
		return fmt.Errorf("user with phone %s already exists", user.Phone)
	}

	result := DB.Create(user)
	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "uni_users_email") {
			return fmt.Errorf("user with email %s already exists", user.Email)
		}
		log.Printf("Error creating user: %T", result.Error)
		return fmt.Errorf("failed to create user")
	}

	// Invalidate potential cache entries
	go func() {
		RedisClient.Del(RedisCtx,
			GetUserCacheKeyByEmail(user.Email),
			GetUserCacheKeyByPhone(user.Phone),
		)
	}()

	return nil
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
