package handlers

import (
	"log"
	"orus/internal/config"
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/utils"
	"orus/internal/validation"
	"time"

	"github.com/gofiber/fiber/v2"
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

	// Invalidate cache before fetching fresh user data
	repositories.InvalidateUserCache(user.ID)

	user, err = repositories.GetUserByID(user.ID)
	if err != nil {
		log.Println("Error fetching updated user:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	// Generate tokens using the centralized JWT utility.
	accessToken, refreshToken, err := utils.GenerateTokens(&models.UserClaims{
		UserID:       user.ID,
		Email:        user.Email,
		Role:         user.Role,
		TokenVersion: user.TokenVersion,
		Permissions:  models.GetDefaultPermissions(user.Role),
	})
	if err != nil {
		log.Println("Error generating tokens:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	// Log the permissions being set (for debugging)
	permissions := models.GetDefaultPermissions(user.Role)
	log.Printf("Setting permissions for user %d with role %s: %v", user.ID, user.Role, permissions)

	setAuthCookies(c, accessToken, refreshToken)

	return c.JSON(fiber.Map{
		"token":         accessToken,
		"refresh_token": refreshToken,
		"user": fiber.Map{
			"id":          user.ID,
			"email":       user.Email,
			"role":        user.Role,
			"permissions": permissions,
		},
	})
}

// RefreshToken handles token refresh requests
func RefreshToken(c *fiber.Ctx) error {
	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Refresh token not provided"})
	}

	// Use the centralized ParseToken.
	_, claims, err := utils.ParseToken(refreshToken)
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

	newAccessToken, newRefreshToken, err := utils.GenerateTokens(&models.UserClaims{
		UserID:       user.ID,
		Email:        user.Email,
		Role:         user.Role,
		TokenVersion: user.TokenVersion,
		Permissions:  models.GetDefaultPermissions(user.Role),
	})
	if err != nil {
		log.Println("Error generating new tokens:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	setAuthCookies(c, newAccessToken, newRefreshToken)

	return c.JSON(fiber.Map{
		"token":         newAccessToken,
		"refresh_token": newRefreshToken,
	})
}

// LogoutUser handles user logout
func LogoutUser(c *fiber.Ctx) error {
	claims, ok := c.Locals("claims").(*models.UserClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid claims"})
	}

	err := repositories.IncrementUserTokenVersion(claims.UserID)
	if err != nil {
		log.Printf("Failed to increment token version for user %d: %v", claims.UserID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to logout"})
	}

	// Clear tokens by setting expired cookies.
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   config.IsProduction(),
		Path:     "/",
		SameSite: "Lax",
	})
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   config.IsProduction(),
		Path:     "/",
		SameSite: "Lax",
	})

	return c.JSON(fiber.Map{"message": "Successfully logged out"})
}

// ChangePassword handles password change requests
func ChangePassword(c *fiber.Ctx) error {
	var input struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	claims, ok := c.Locals("claims").(*models.UserClaims)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid claims"})
	}
	userID := claims.UserID

	user, err := repositories.GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get user"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.OldPassword)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid old password"})
	}

	// Validate new password: minimum length and special character requirement.
	if len(input.NewPassword) < 8 || !validation.HasSpecialChar(input.NewPassword) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Password must be at least 8 characters and contain special characters"})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to hash password"})
	}

	user.Password = string(hashedPassword)
	user.TokenVersion++ // Invalidate existing tokens.
	if err := repositories.DB.Save(user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update password"})
	}

	repositories.InvalidateUserCache(user.ID)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Password changed successfully"})
}

// Helper functions
func getUserByIdentifier(email, phone string) (*models.User, error) {
	if email != "" {
		return repositories.GetUserByEmail(email)
	}
	return repositories.GetUserByPhone(phone)
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
