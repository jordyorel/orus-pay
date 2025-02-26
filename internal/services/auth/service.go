package auth

import (
	"errors"
	"log"
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/utils"
	"orus/internal/validation"

	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	Login(email, phone, password string) (*models.User, string, string, error)
	RefreshTokens(refreshToken string) (string, string, error)
	Logout(userID uint) error
	ChangePassword(userID uint, oldPassword, newPassword string) error
}

type service struct {
	userRepo repositories.UserRepository
}

func NewService(userRepo repositories.UserRepository) Service {
	return &service{
		userRepo: userRepo,
	}
}

func (s *service) Login(email, phone, password string) (*models.User, string, string, error) {
	// Get user by email or phone
	user, err := s.getUserByIdentifier(email, phone)
	if err != nil {
		log.Printf("Login failed: User not found for identifier: %s", email+phone)
		return nil, "", "", errors.New("invalid credentials")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		log.Printf("Login failed: Incorrect password for user ID: %d", user.ID)
		return nil, "", "", errors.New("invalid credentials")
	}

	// Generate tokens
	accessToken, refreshToken, err := utils.GenerateTokens(&models.UserClaims{
		UserID:       user.ID,
		Email:        user.Email,
		Role:         user.Role,
		TokenVersion: user.TokenVersion,
		Permissions:  models.GetDefaultPermissions(user.Role),
	})
	if err != nil {
		log.Println("Error generating tokens:", err)
		return nil, "", "", errors.New("error generating tokens")
	}

	return user, accessToken, refreshToken, nil
}

func (s *service) RefreshTokens(refreshToken string) (string, string, error) {
	_, claims, err := utils.ParseToken(refreshToken)
	if err != nil {
		return "", "", errors.New("invalid refresh token")
	}

	user, err := s.userRepo.GetByID(claims.UserID)
	if err != nil {
		return "", "", errors.New("user not found")
	}

	if user.TokenVersion != claims.TokenVersion {
		return "", "", errors.New("token version mismatch")
	}

	return utils.GenerateTokens(&models.UserClaims{
		UserID:       user.ID,
		Email:        user.Email,
		Role:         user.Role,
		TokenVersion: user.TokenVersion,
		Permissions:  models.GetDefaultPermissions(user.Role),
	})
}

func (s *service) Logout(userID uint) error {
	return s.userRepo.IncrementTokenVersion(userID)
}

func (s *service) ChangePassword(userID uint, oldPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return errors.New("failed to get user")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return errors.New("invalid old password")
	}

	if len(newPassword) < 8 || !validation.HasSpecialChar(newPassword) {
		return errors.New("password must be at least 8 characters and contain special characters")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("failed to hash password")
	}

	user.Password = string(hashedPassword)
	user.TokenVersion++ // Invalidate existing tokens

	if err := s.userRepo.Update(user); err != nil {
		return errors.New("failed to update password")
	}

	return nil
}

func (s *service) getUserByIdentifier(email, phone string) (*models.User, error) {
	if email != "" {
		return s.userRepo.GetByEmail(email)
	}
	return s.userRepo.GetByPhone(phone)
}
