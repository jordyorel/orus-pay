package handlers

import (
	"errors"
	"log"
	"orus/internal/config"
	"orus/internal/models"
	"orus/internal/services/auth"
	"orus/internal/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type AuthHandler struct {
	authService   auth.Service
	refreshSecret string
}

func NewAuthHandler(authService auth.Service, refreshSecret string) *AuthHandler {
	return &AuthHandler{
		authService:   authService,
		refreshSecret: refreshSecret,
	}
}

// LoginUser handles user authentication and returns JWT tokens
func (h *AuthHandler) LoginUser(c *fiber.Ctx) error {
	var input struct {
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate input
	if (input.Email == "" && input.Phone == "") || input.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email/phone and password are required",
		})
	}

	user, accessToken, refreshToken, err := h.authService.Login(input.Email, input.Phone, input.Password)
	if err != nil {
		if errors.Is(err, auth.ErrMFARequired) {
			return c.JSON(fiber.Map{
				"mfa_required": true,
				"user_id":      user.ID,
			})
		}
		if err.Error() == "invalid credentials" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid email or password",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Authentication failed",
		})
	}

	h.setAuthCookies(c, accessToken, refreshToken)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"access_token":  accessToken,
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
	// First try to get token from cookies
	refreshToken := c.Cookies("refresh_token")

	// If not in cookies, try request body
	if refreshToken == "" {
		var input struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.BodyParser(&input); err != nil {
			return utils.Unauthorized(c, "Refresh token not provided")
		}
		refreshToken = input.RefreshToken
	}

	// Validate refresh token
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

	// Increment token version to invalidate all existing tokens
	if err := h.authService.Logout(claims.UserID); err != nil {
		return utils.InternalError(c, "Failed to logout")
	}

	// Clear cookies
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   config.IsProduction(),
		Path:     "/",
	})
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   config.IsProduction(),
		Path:     "/",
	})

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

// VerifyOTP completes login after MFA code validation
func (h *AuthHandler) VerifyOTP(c *fiber.Ctx) error {
	var input struct {
		UserID uint   `json:"user_id"`
		Code   string `json:"code"`
	}
	if err := c.BodyParser(&input); err != nil {
		return utils.BadRequest(c, "Invalid request body")
	}

	user, access, refresh, err := h.authService.VerifyOTP(input.UserID, input.Code)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}

	h.setAuthCookies(c, access, refresh)

	return utils.Success(c, fiber.Map{
		"access_token":  access,
		"refresh_token": refresh,
		"user": fiber.Map{
			"id":          user.ID,
			"email":       user.Email,
			"role":        user.Role,
			"permissions": models.GetDefaultPermissions(user.Role),
		},
	})
}

// GetTokenVersion handles getting the token version of a user
func (h *AuthHandler) GetTokenVersion(c *fiber.Ctx) error {
	userID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	version, err := h.authService.GetUserTokenVersion(uint(userID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to get token version",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"user_id":       userID,
		"token_version": version,
	})
}

// Add this new handler to debug the token
func (h *AuthHandler) DebugToken(c *fiber.Ctx) error {
	// Get the token from the Authorization header
	authHeader := c.Get("Authorization")
	if authHeader == "" || len(strings.Split(authHeader, " ")) != 2 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing or invalid Authorization header",
		})
	}

	tokenString := strings.Split(authHeader, " ")[1]

	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &models.UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.refreshSecret), nil
	})

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid token",
			"details": err.Error(),
		})
	}

	claims, ok := token.Claims.(*models.UserClaims)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid token claims",
		})
	}

	// Get the current token version from the database
	currentVersion, err := h.authService.GetUserTokenVersion(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to get current token version",
			"details": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"token_claims":    claims,
		"current_version": currentVersion,
		"is_valid":        claims.TokenVersion == currentVersion,
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
