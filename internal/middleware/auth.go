package middleware

import (
	"log"
	"strings"
	"time"

	"orus/internal/config"
	"orus/internal/models"
	"orus/internal/repositories"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(c *fiber.Ctx) error {
	// Try to get the token from the cookie first
	tokenString := c.Cookies("access_token")
	if tokenString == "" {
		// If the cookie is missing, try to get the token from the Authorization header
		authHeader := c.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			log.Println("🔴 Authorization cookie and header are missing")
			log.Printf("🔴 Request headers: %v", c.GetReqHeaders())
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}
	}

	// Parse and validate the token
	token, err := jwt.ParseWithClaims(tokenString, &models.UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.GetEnv("JWT_SECRET", "")), nil
	})

	if err != nil {
		log.Println("🔴 Invalid token:", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
	}

	claims, ok := token.Claims.(*models.UserClaims)
	if !ok {
		log.Println("🔴 Invalid token claims type")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token claims"})
	}

	// Validate expiration
	if claims.ExpiresAt.Before(time.Now()) {
		log.Println("🔴 Token has expired")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Token has expired"})
	}

	// Fetch user and validate token version
	user, err := repositories.GetUserByID(claims.UserID)
	if err != nil {
		log.Println("🔴 Error fetching user:", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user"})
	}

	if claims.TokenVersion != user.TokenVersion {
		log.Println("🔴 Token version mismatch")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Token has been invalidated"})
	}

	// Store the entire claims object in context
	c.Locals("claims", claims)
	c.Locals("userID", claims.UserID)
	c.Locals("role", claims.Role)
	c.Locals("permissions", claims.Permissions)

	return c.Next()
}

func AdminAuthMiddleware(c *fiber.Ctx) error {
	claims, ok := c.Locals("claims").(*models.UserClaims)
	if !ok {
		log.Println("🔴 Claims not found in locals")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid claims"})
	}

	if !claims.HasPermission(models.PermissionReadAdmin) {
		log.Println("🔴 Insufficient permissions")
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Insufficient permissions"})
	}

	log.Println("✅ Admin authenticated successfully")
	return c.Next()
}

// HasPermission middleware checks if the user has a specific permission
func HasPermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals("claims").(*models.UserClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid claims"})
		}

		if !claims.HasPermission(permission) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Insufficient permissions"})
		}

		return c.Next()
	}
}
