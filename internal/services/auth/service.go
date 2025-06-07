// Package auth provides authentication and authorization services.
// It handles user authentication, token management, and permission validation.
package auth

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/repositories/cache"
	"orus/internal/validation"

	"log"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ErrMFARequired indicates that multi-factor authentication is needed
var ErrMFARequired = errors.New("mfa_required")

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

	// VerifyOTP completes login when MFA is enabled
	VerifyOTP(userID uint, code string) (*models.User, string, string, error)
}

type service struct {
	userRepo      repositories.UserRepository
	jwtSecret     string
	refreshSecret string
	cache         *cache.CacheService
}

func NewService(userRepo repositories.UserRepository, jwtSecret, refreshSecret string, cacheSvc *cache.CacheService) Service {
	return &service{
		userRepo:      userRepo,
		jwtSecret:     jwtSecret,
		refreshSecret: refreshSecret,
		cache:         cacheSvc,
	}
}

func (s *service) Login(email, phone, password string) (*models.User, string, string, error) {
	// Get user by email or phone
	var user *models.User
	var err error

	if email != "" {
		user, err = s.userRepo.GetByEmail(email)
	} else {
		user, err = s.userRepo.GetByPhone(phone)
	}

	if err != nil {
		return nil, "", "", err
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, "", "", errors.New("invalid credentials")
	}

	// If MFA is enabled, generate OTP and return special error
	if user.TwoFactorEnabled {
		if _, err := s.generateOTP(user.ID); err != nil {
			return nil, "", "", err
		}
		return user, "", "", ErrMFARequired
	}

	// After successful login
	log.Printf("Initial token version: %d", user.TokenVersion)
	log.Printf("User ID before login: %d", user.ID)

	if err := s.userRepo.IncrementTokenVersion(user.ID); err != nil {
		return nil, "", "", err
	}

	// Verify the increment
	updatedUser, err := s.userRepo.GetByID(user.ID)
	if err != nil {
		return nil, "", "", err
	}
	log.Printf("New token version: %d", updatedUser.TokenVersion)
	log.Printf("User ID after increment: %d", updatedUser.ID)

	// Generate new tokens
	accessToken, err := s.generateAccessToken(updatedUser)
	if err != nil {
		return nil, "", "", err
	}
	log.Printf("Generated token with version: %d for user ID: %d", updatedUser.TokenVersion, updatedUser.ID)

	refreshToken, err := s.generateRefreshToken(updatedUser)
	if err != nil {
		return nil, "", "", err
	}

	return updatedUser, accessToken, refreshToken, nil
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

func (s *service) generateTokens(user *models.User) (string, string, error) {
	// Create access token with JWT_SECRET
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return "", "", err
	}

	// Create refresh token with REFRESH_SECRET
	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *service) generateAccessToken(user *models.User) (string, error) {
	claims := &models.UserClaims{
		UserID:       user.ID,
		Email:        user.Email,
		Role:         user.Role,
		Permissions:  models.GetDefaultPermissions(user.Role),
		TokenType:    "access",
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *service) generateRefreshToken(user *models.User) (string, error) {
	claims := &models.UserClaims{
		UserID:       user.ID,
		TokenType:    "refresh",
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 7)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.refreshSecret))
}

func (s *service) GetUserByID(userID uint) (*models.User, error) {
	return s.userRepo.GetByID(userID)
}

func (s *service) GenerateTokens(user *models.User) (string, string, error) {
	// Reuse the existing generateTokens method
	return s.generateTokens(user)
}

func (s *service) GetUserTokenVersion(userID uint) (int, error) {
	log.Printf("Getting token version for user ID: %d", userID)
	user, err := s.GetUserByID(userID)
	if err != nil {
		log.Printf("Error getting token version for user %d: %v", userID, err)
		return 0, err
	}
	log.Printf("Retrieved token version %d for user %d", user.TokenVersion, userID)
	return user.TokenVersion, nil
}

// generateOTP creates a 6 digit code and stores it in cache
func (s *service) generateOTP(userID uint) (string, error) {
	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	key := fmt.Sprintf("otp:%d", userID)
	if err := s.cache.SetWithTTL(context.Background(), key, code, 5*time.Minute); err != nil {
		return "", err
	}
	log.Printf("OTP for user %d: %s", userID, code)
	return code, nil
}

// VerifyOTP checks the code and returns tokens if valid
func (s *service) VerifyOTP(userID uint, code string) (*models.User, string, string, error) {
	key := fmt.Sprintf("otp:%d", userID)
	var stored string
	found, err := s.cache.Get(context.Background(), key, &stored)
	if err != nil || !found || stored != code {
		return nil, "", "", errors.New("invalid otp")
	}
	_ = s.cache.Delete(context.Background(), key)

	if err := s.userRepo.IncrementTokenVersion(userID); err != nil {
		return nil, "", "", err
	}
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, "", "", err
	}

	access, refresh, err := s.generateTokens(user)
	if err != nil {
		return nil, "", "", err
	}

	return user, access, refresh, nil
}
