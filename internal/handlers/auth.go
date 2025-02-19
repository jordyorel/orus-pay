package handlers

import (
	"log"
	"orus/internal/config"
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/utils"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// LoginUser handles user login requests
func LoginUser(c *fiber.Ctx) error {
	var loginDetails struct {
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&loginDetails); err != nil {
		log.Println("Error parsing login request:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request format"})
	}

	if (loginDetails.Email == "" && loginDetails.Phone == "") || loginDetails.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Email or phone and password are required"})
	}

	user, err := getUserByIdentifier(loginDetails.Email, loginDetails.Phone)
	if err != nil {
		log.Printf("Login failed: User not found for identifier: %s", loginDetails.Email+loginDetails.Phone)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginDetails.Password)); err != nil {
		log.Printf("Login failed: Incorrect password for user ID: %d", user.ID)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid credentials"})
	}

	// if err := repositories.IncrementUserTokenVersion(user.ID); err != nil {
	// 	log.Println("Error incrementing token version:", err)
	// 	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	// }

	// Invalidate cache before fetching fresh user data
	repositories.InvalidateUserCache(user.ID)

	user, err = repositories.GetUserByID(user.ID)
	if err != nil {
		log.Println("Error fetching updated user:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	accessToken, refreshToken, err := generateTokens(user)
	if err != nil {
		log.Println("Error generating tokens:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	setAuthCookies(c, accessToken, refreshToken)

	return c.JSON(fiber.Map{
		"token":         accessToken,
		"refresh_token": refreshToken,
		"user": fiber.Map{
			"id":          user.ID,
			"email":       user.Email,
			"role":        user.Role,
			"permissions": models.GetDefaultPermissions(user.Role),
		},
	})
}

// RefreshToken handles token refresh requests
func RefreshToken(c *fiber.Ctx) error {
	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Refresh token not provided"})
	}

	claims, err := parseToken(refreshToken)
	if err != nil {
		log.Println("Error parsing refresh token:", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid refresh token"})
	}

	user, err := repositories.GetUserByID(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "User not found"})
	}

	if user.TokenVersion != claims.TokenVersion {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Token version mismatch"})
	}

	accessToken, refreshToken, err := generateTokens(user)
	if err != nil {
		log.Println("Error generating tokens:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	setAuthCookies(c, accessToken, refreshToken)

	return c.JSON(fiber.Map{
		"token":         accessToken,
		"refresh_token": refreshToken,
	})
}

// LogoutUser handles user logout
func LogoutUser(c *fiber.Ctx) error {
	claims, ok := c.Locals("claims").(*models.UserClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid claims"})
	}

	// Invalidate all tokens by incrementing the token version
	err := repositories.IncrementUserTokenVersion(claims.UserID)
	if err != nil {
		log.Printf("🔴 Failed to increment token version: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to logout"})
	}

	// Clear the access_token cookie with proper settings
	cookie := &fiber.Cookie{
		Name:     "access_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour), // Expire immediately
		HTTPOnly: true,
		Secure:   config.IsProduction(), // Match login settings
		SameSite: "Lax",                 // Match login settings
		Path:     "/",                   // Explicitly set path to match login
	}
	c.Cookie(cookie)

	return c.JSON(fiber.Map{"message": "Successfully logged out"})
}

// ChangePassword handles password change requests
func ChangePassword(c *fiber.Ctx) error {
	var input struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get user from claims
	claims := c.Locals("claims").(*models.UserClaims)
	userID := claims.UserID

	// Get user from database
	user, err := repositories.GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get user",
		})
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.OldPassword)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid old password",
		})
	}

	// Validate new password
	if len(input.NewPassword) < 8 || !utils.HasSpecialChar(input.NewPassword) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password must be at least 8 characters and contain special characters",
		})
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to hash password",
		})
	}

	// Update password and increment token version to invalidate existing tokens
	user.Password = string(hashedPassword)
	user.TokenVersion++
	if err := repositories.DB.Save(user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update password",
		})
	}

	// Invalidate user cache
	repositories.InvalidateUserCache(user.ID)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Password changed successfully",
	})
}

// Helper functions
func getUserByIdentifier(email, phone string) (*models.User, error) {
	if email != "" {
		return repositories.GetUserByEmail(email)
	}
	return repositories.GetUserByPhone(phone)
}

func generateTokens(user *models.User) (string, string, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("Missing JWT_SECRET environment variable")
	}

	now := time.Now()
	accessClaims := &models.UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatUint(uint64(user.ID), 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			Issuer:    "orus-api",
		},
		UserID:       user.ID,
		Email:        user.Email,
		Role:         user.Role,
		Permissions:  models.GetDefaultPermissions(user.Role),
		TokenVersion: user.TokenVersion,
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(jwtSecret))
	if err != nil {
		return "", "", err
	}

	refreshClaims := &models.UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatUint(uint64(user.ID), 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
			Issuer:    "orus-api",
		},
		UserID:       user.ID,
		Email:        user.Email,
		Role:         user.Role,
		Permissions:  models.GetDefaultPermissions(user.Role),
		TokenVersion: user.TokenVersion,
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(jwtSecret))
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func setAuthCookies(c *fiber.Ctx, accessToken, refreshToken string) {
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		HTTPOnly: true,
		Secure:   config.IsProduction(),
		Path:     "/",
		SameSite: "Strict",
	})
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HTTPOnly: true,
		Secure:   config.IsProduction(),
		Path:     "/",
		SameSite: "Strict",
	})
}

func parseToken(tokenString string) (*models.UserClaims, error) {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("Missing JWT_SECRET environment variable")
	}

	token, err := jwt.ParseWithClaims(tokenString, &models.UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.ErrUnauthorized
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*models.UserClaims)
	if !ok || !token.Valid {
		return nil, fiber.ErrUnauthorized
	}

	return claims, nil
}
