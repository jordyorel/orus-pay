package repositories

import (
	"context"
	"fmt"
	"log"
	"orus/internal/models"
	"strconv"
	"time"
)

const (
	userCacheExpiration = 24 * time.Hour
)

func GetUserByEmail(email string) (*models.User, error) {
	// Try cache first
	key := CacheService.GenerateKey("user", "email", email)
	var cachedUser models.User
	if found, _ := CacheService.Get(context.Background(), key, &cachedUser); found {
		log.Printf("Cache hit for user email: %s", email)
		return &cachedUser, nil
	}
	if err := DB.Where("email = ?", email).First(&cachedUser).Error; err != nil {
		return nil, err
	}

	// Cache result
	CacheService.SetWithTTL(context.Background(), key, cachedUser, 24*time.Hour)

	return &cachedUser, nil
}

func GetUserByID(userID uint) (*models.User, error) {
	key := CacheService.GenerateKey("user", "id", strconv.FormatUint(uint64(userID), 10))
	var cachedUser models.User
	found, _ := CacheService.Get(context.Background(), key, &cachedUser)
	if found {
		log.Printf("Cache hit for user ID: %d", userID)
		return &cachedUser, nil
	}
	if err := DB.Where("id = ?", userID).First(&cachedUser).Error; err != nil {
		return nil, err
	}

	// Cache result
	CacheService.SetWithTTL(context.Background(), key, cachedUser, 24*time.Hour)

	return &cachedUser, nil
}

func GetUserByPhone(phone string) (*models.User, error) {
	key := CacheService.GenerateKey("user", "phone", phone)
	var cachedUser models.User
	found, _ := CacheService.Get(context.Background(), key, &cachedUser)
	if found {
		log.Printf("Cache hit for user phone: %s", phone)
		return &cachedUser, nil
	}

	var user models.User
	result := DB.Where("phone = ?", phone).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}

	// Cache the result
	CacheService.SetWithTTL(context.Background(), key, user, 24*time.Hour)

	return &user, nil
}

func CreateUser(user *models.User) (*models.User, *models.QRCode, error) {
	tx := DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(user).Error; err != nil {
		tx.Rollback()
		return nil, nil, err
	}

	// Remove any QR code creation here - we handle that in the service now

	if err := tx.Commit().Error; err != nil {
		return nil, nil, err
	}

	return user, nil, nil
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

// Instead, implement the function at the package level
func GetUserTransactionsPaginated(userID uint, limit, offset int) ([]models.Transaction, int64, error) {
	var transactions []models.Transaction
	var total int64

	// Count total transactions
	if err := DB.Model(&models.Transaction{}).
		Where("sender_id = ? OR receiver_id = ?", userID, userID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated transactions
	if err := DB.Where("sender_id = ? OR receiver_id = ?", userID, userID).
		Order("transaction_id DESC").
		Limit(limit).
		Offset(offset).
		Find(&transactions).Error; err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}

// Example implementation
func (r *userRepository) GetBalance(userID uint) (float64, error) {
	var user models.User
	err := r.db.First(&user, userID).Error
	if err != nil {
		return 0, err
	}
	return user.Balance, nil
}

func (r *userRepository) UpdateBalance(userID uint, newBalance float64) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("balance", newBalance).Error
}

// Add this function to handle user cache invalidation
func InvalidateUserCache(userID uint) error {
	// Generate keys for all user cache entries
	idKey := CacheService.GenerateKey("user", "id", userID)

	// Delete the cache entries
	if err := CacheService.Delete(context.Background(), idKey); err != nil {
		return err
	}

	// Log the invalidation
	log.Printf("Invalidated cache for user ID: %d", userID)
	return nil
}
