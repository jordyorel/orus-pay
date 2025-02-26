package user

import (
	"errors"
	"orus/internal/models"
	"orus/internal/repositories"

	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	GetByID(id uint) (*models.User, error)
	Create(input *models.CreateUserInput) (*models.User, error)
	Update(user *models.User) error
	Delete(id uint) error
	ChangePassword(userID uint, oldPassword, newPassword string) error
	GetTransactions(userID uint, page, limit int) ([]models.Transaction, int64, error)
}

type service struct {
	repo repositories.UserRepository
}

func NewService(repo repositories.UserRepository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) GetByID(id uint) (*models.User, error) {
	return s.repo.GetByID(id)
}

func (s *service) Create(input *models.CreateUserInput) (*models.User, error) {
	if input.Email == "" {
		return nil, errors.New("email is required")
	}

	// Check if user already exists
	existingUser, _ := s.repo.GetByEmail(input.Email)
	if existingUser != nil {
		return nil, errors.New("user with this email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	// Create user
	user := &models.User{
		Name:     input.Name,
		Email:    input.Email,
		Phone:    input.Phone,
		Password: string(hashedPassword),
		Role:     input.Role,
		Status:   "active",
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *service) Update(user *models.User) error {
	return s.repo.Update(user)
}

func (s *service) Delete(id uint) error {
	return s.repo.Delete(id)
}

func (s *service) ChangePassword(userID uint, oldPassword, newPassword string) error {
	user, err := s.repo.GetByID(userID)
	if err != nil {
		return errors.New("user not found")
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("incorrect password")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("failed to hash password")
	}

	user.Password = string(hashedPassword)
	user.TokenVersion++ // Invalidate existing tokens

	return s.repo.Update(user)
}

func (s *service) GetTransactions(userID uint, page, limit int) ([]models.Transaction, int64, error) {
	offset := (page - 1) * limit
	return repositories.GetUserTransactionsPaginated(userID, limit, offset)
}
