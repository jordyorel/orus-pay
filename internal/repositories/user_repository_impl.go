package repositories

import (
	"orus/internal/models"

	"log"

	"context"

	"orus/internal/repositories/cache"

	"gorm.io/gorm"
)

type userRepository struct {
	db    *gorm.DB
	cache *cache.CacheService
}

// NewUserRepository creates a new instance of UserRepository
func NewUserRepository(db *gorm.DB, cache *cache.CacheService) UserRepository {
	return &userRepository{
		db:    db,
		cache: cache,
	}
}

func (r *userRepository) Create(user *models.User) error {
	result := r.db.Create(user)
	if result.Error != nil {
		return ErrDatabaseOperation
	}
	return nil
}

func (r *userRepository) GetByID(id uint) (*models.User, error) {
	log.Printf("GetByID called for user ID: %d", id)

	// Try cache first
	key := r.cache.GenerateKey("user", "id", id)
	log.Printf("Checking cache with key: %s", key)
	if user, err := r.cache.GetUser(context.Background(), key); err == nil {
		log.Printf("Cache hit for user ID: %d", id)
		return user, nil
	}

	log.Printf("Cache miss for user ID: %d, querying database", id)
	// Cache miss - proceed to database
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		log.Printf("Database error for user ID %d: %v", id, err)
		if err == gorm.ErrRecordNotFound {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	log.Printf("Found user in database: ID=%d, Email=%s", user.ID, user.Email)
	// Cache the result
	if err := r.cache.CacheUser(context.Background(), &user); err != nil {
		log.Printf("Failed to cache user: %v", err)
	}

	return &user, nil
}

func (r *userRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	result := r.db.Where("email = ?", email).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, ErrUserNotFound
		}
		return nil, ErrDatabaseOperation
	}
	return &user, nil
}

func (r *userRepository) GetByPhone(phone string) (*models.User, error) {
	var user models.User
	result := r.db.Where("phone = ?", phone).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, ErrUserNotFound
		}
		return nil, ErrDatabaseOperation
	}
	return &user, nil
}

func (r *userRepository) Update(user *models.User) error {
	result := r.db.Save(user)
	if result.Error != nil {
		return ErrDatabaseOperation
	}

	// Invalidate cache after update
	log.Printf("Invalidating cache for updated user ID: %d", user.ID)
	if err := r.cache.InvalidateUser(context.Background(), user.ID); err != nil {
		log.Printf("Warning: Failed to invalidate user cache: %v", err)
	}

	return nil
}

func (r *userRepository) Delete(id uint) error {
	result := r.db.Delete(&models.User{}, id)
	if result.Error != nil {
		return ErrDatabaseOperation
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *userRepository) IncrementTokenVersion(userID uint) error {
	// Update token version in database
	if err := r.db.Model(&models.User{}).Where("id = ?", userID).
		UpdateColumn("token_version", gorm.Expr("token_version + 1")).Error; err != nil {
		return err
	}

	// Add debug logging
	log.Printf("Invalidating cache for user ID: %d", userID)
	if err := InvalidateUserCache(userID); err != nil {
		log.Printf("Cache invalidation error: %v", err)
	}

	log.Printf("Incremented and invalidated cache for user ID: %d", userID)
	return nil
}

func (r *userRepository) List(offset, limit int) ([]*models.User, int64, error) {
	var users []*models.User
	var total int64

	// Get total count
	if err := r.db.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, ErrDatabaseOperation
	}

	// Get users with pagination
	result := r.db.Offset(offset).Limit(limit).Find(&users)
	if result.Error != nil {
		return nil, 0, ErrDatabaseOperation
	}

	return users, total, nil
}

func (r *userRepository) UpdatePassword(userID uint, hashedPassword string) error {
	result := r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("password", hashedPassword)
	if result.Error != nil {
		return ErrDatabaseOperation
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *userRepository) UpdateStatus(userID uint, status string) error {
	result := r.db.Model(&models.User{}).
		Where("id = ?", userID).
		Update("status", status)
	if result.Error != nil {
		return ErrDatabaseOperation
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}
