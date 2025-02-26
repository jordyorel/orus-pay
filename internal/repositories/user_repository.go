package repositories

import (
	"errors"
	"orus/internal/models"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrEmailTaken        = errors.New("email already taken")
	ErrPhoneTaken        = errors.New("phone number already taken")
	ErrInvalidUserData   = errors.New("invalid user data")
	ErrDatabaseOperation = errors.New("database operation failed")
)

// UserRepository defines the interface for user-related database operations
type UserRepository interface {
	// Create creates a new user in the database
	Create(user *models.User) error

	// GetByID retrieves a user by their ID
	GetByID(id uint) (*models.User, error)

	// GetByEmail retrieves a user by their email address
	GetByEmail(email string) (*models.User, error)

	// GetByPhone retrieves a user by their phone number
	GetByPhone(phone string) (*models.User, error)

	// Update updates an existing user's information
	Update(user *models.User) error

	// Delete removes a user from the database
	Delete(id uint) error

	// IncrementTokenVersion increments the user's token version
	IncrementTokenVersion(userID uint) error

	// List retrieves users with pagination
	List(offset, limit int) ([]*models.User, int64, error)

	// UpdatePassword updates the user's password
	UpdatePassword(userID uint, hashedPassword string) error

	// UpdateStatus updates the user's status
	UpdateStatus(userID uint, status string) error
}

// Implementation will be in user_repository_impl.go
