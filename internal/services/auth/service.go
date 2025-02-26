// Package auth provides authentication and authorization services.
// It handles user authentication, token management, and permission validation.
package auth

import (
	"errors"
	"log"
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/validation"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Service defines the interface for authentication operations.
// It provides methods for user authentication, token management,
// and session handling.
type Service interface {
	// Login authenticates a user and returns access and refresh tokens
	Login(email, phone, password string) (*models.User, string, string, error)

	// RefreshTokens generates new access and refresh tokens
	RefreshTokens(refreshToken string) (string, string, error)

	// Logout invalidates a user's current session
	Logout(userID uint) error

	// GetUserTokenVersion returns the current token version for a user
	GetUserTokenVersion(userID uint) (int, error)

	// ChangePassword updates a user's password after validating the old password
	// Returns error if old password is invalid or new password doesn't meet requirements
	ChangePassword(userID uint, oldPassword, newPassword string) error

	// GetUserByID retrieves a user by their ID
	GetUserByID(userID uint) (*models.User, error)

	// GenerateTokens creates new access and refresh tokens for a user
	GenerateTokens(user *models.User) (string, string, error)
}

type service struct {
	userRepo      repositories.UserRepository
	jwtSecret     string
	refreshSecret string
}

func NewService(userRepo repositories.UserRepository, jwtSecret, refreshSecret string) Service {
	return &service{
		userRepo:      userRepo,
		jwtSecret:     jwtSecret,
		refreshSecret: refreshSecret,
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
	accessToken, refreshToken, err := s.generateTokens(user)
	if err != nil {
		log.Println("Error generating tokens:", err)
		return nil, "", "", errors.New("error generating tokens")
	}

	return user, accessToken, refreshToken, nil
}

func (s *service) RefreshTokens(refreshToken string) (string, string, error) {
	// Parse refresh token with REFRESH_SECRET
	token, err := jwt.ParseWithClaims(refreshToken, &models.UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.refreshSecret), nil // Use refresh secret here
	})
	if err != nil {
		return "", "", errors.New("invalid refresh token")
	}

	// Rest of the refresh token validation...
	claims, ok := token.Claims.(*models.UserClaims)
	if !ok {
		return "", "", errors.New("invalid token claims")
	}

	user, err := s.GetUserByID(claims.UserID)
	if err != nil {
		return "", "", errors.New("user not found")
	}

	if claims.TokenVersion != user.TokenVersion {
		return "", "", errors.New("token version mismatch")
	}

	return s.generateTokens(user)
}

func (s *service) Logout(userID uint) error {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return err
	}

	// Increment token version
	user.TokenVersion++

	// Update user in database
	if err := s.userRepo.Update(user); err != nil {
		return err
	}

	return nil
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

func (s *service) generateTokens(user *models.User) (string, string, error) {
	// Create access token with JWT_SECRET
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, models.UserClaims{
		UserID:       user.ID,
		Email:        user.Email,
		Role:         user.Role,
		Permissions:  models.GetDefaultPermissions(user.Role),
		TokenType:    "access",
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
		},
	})

	// Sign access token with JWT_SECRET
	accessTokenString, err := accessToken.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", "", err
	}

	// Create refresh token with REFRESH_SECRET
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, models.UserClaims{
		UserID:       user.ID,
		TokenType:    "refresh",
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 7)),
		},
	})

	// Sign refresh token with REFRESH_SECRET (not the fixed secret)
	refreshTokenString, err := refreshToken.SignedString([]byte(s.refreshSecret))
	if err != nil {
		return "", "", err
	}

	return accessTokenString, refreshTokenString, nil
}

func (s *service) GetUserByID(userID uint) (*models.User, error) {
	return s.userRepo.GetByID(userID)
}

func (s *service) GenerateTokens(user *models.User) (string, string, error) {
	// Reuse the existing generateTokens method
	return s.generateTokens(user)
}

func (s *service) GetUserTokenVersion(userID uint) (int, error) {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return 0, err
	}
	return user.TokenVersion, nil
}
