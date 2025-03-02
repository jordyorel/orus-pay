package repositories

import (
	"orus/internal/models"

	"log"

	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new instance of UserRepository
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		db: db,
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
	var user models.User
	result := r.db.First(&user, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, ErrUserNotFound
		}
		return nil, ErrDatabaseOperation
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
