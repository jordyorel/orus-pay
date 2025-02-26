package handlers

import (
	"log"
	"orus/internal/config"
	"orus/internal/models"
	"orus/internal/services/auth"
	"orus/internal/utils"
	"time"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	authService auth.Service
}

func NewAuthHandler(authService auth.Service) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// LoginUser handles user login requests
func (h *AuthHandler) LoginUser(c *fiber.Ctx) error {
	var input struct {
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&input); err != nil {
		log.Printf("Error parsing login request: %v", err)
		return utils.BadRequest(c, "Invalid request format")
	}

	// Validate input
	if (input.Email == "" && input.Phone == "") || input.Password == "" {
		return utils.BadRequest(c, "Email/phone and password are required")
	}

	// Attempt login
	user, accessToken, refreshToken, err := h.authService.Login(input.Email, input.Phone, input.Password)
	if err != nil {
		log.Printf("Login failed: %v", err)
		return utils.Unauthorized(c, "Invalid credentials")
	}

	// Set auth cookies
	h.setAuthCookies(c, accessToken, refreshToken)

	// Return success response
	return utils.Success(c, fiber.Map{
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
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		return utils.Unauthorized(c, "Refresh token not provided")
	}

	// Attempt to refresh tokens
	newAccessToken, newRefreshToken, err := h.authService.RefreshTokens(refreshToken)
	if err != nil {
		log.Printf("Token refresh failed: %v", err)
		return utils.Unauthorized(c, "Invalid refresh token")
	}

	// Set new auth cookies
	h.setAuthCookies(c, newAccessToken, newRefreshToken)

	return utils.Success(c, fiber.Map{
		"token":         newAccessToken,
		"refresh_token": newRefreshToken,
	})
}

// LogoutUser handles user logout
func (h *AuthHandler) LogoutUser(c *fiber.Ctx) error {
	claims, ok := c.Locals("claims").(*models.UserClaims)
	if !ok {
		return utils.Unauthorized(c, "Invalid claims")
	}

	if err := h.authService.Logout(claims.UserID); err != nil {
		log.Printf("Logout failed for user %d: %v", claims.UserID, err)
		return utils.InternalError(c, "Failed to logout")
	}

	// Clear auth cookies
	h.clearAuthCookies(c)

	return utils.Success(c, fiber.Map{
		"message": "Successfully logged out",
	})
}

// ChangePassword handles password change requests
func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	var input struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	if err := c.BodyParser(&input); err != nil {
		return utils.BadRequest(c, "Invalid request body")
	}

	claims, ok := c.Locals("claims").(*models.UserClaims)
	if !ok {
		return utils.Unauthorized(c, "Invalid claims")
	}

	if err := h.authService.ChangePassword(claims.UserID, input.OldPassword, input.NewPassword); err != nil {
		log.Printf("Password change failed for user %d: %v", claims.UserID, err)
		return utils.BadRequest(c, err.Error())
	}

	return utils.Success(c, fiber.Map{
		"message": "Password changed successfully",
	})
}

// Helper methods

func (h *AuthHandler) setAuthCookies(c *fiber.Ctx, accessToken, refreshToken string) {
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		HTTPOnly: true,
		Secure:   config.IsProduction(),
		Path:     "/",
		SameSite: "Strict",
		MaxAge:   15 * 60, // 15 minutes
	})

	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		HTTPOnly: true,
		Secure:   config.IsProduction(),
		Path:     "/",
		SameSite: "Strict",
		MaxAge:   7 * 24 * 60 * 60, // 7 days
	})
}

func (h *AuthHandler) clearAuthCookies(c *fiber.Ctx) {
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
}
