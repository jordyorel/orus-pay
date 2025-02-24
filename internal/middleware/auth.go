package middleware

import (
	"log"
	"strings"

	"orus/internal/models"
	"orus/internal/utils"

	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware verifies the JWT token from the Authorization header.
func AuthMiddleware(c *fiber.Ctx) error {
	tokenString := c.Get("Authorization")
	// Try cookie if header is empty
	if tokenString == "" {
		tokenString = c.Cookies("access_token")
	} else {
		// Remove Bearer prefix if present
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	}

	if tokenString == "" {
		return utils.Unauthorized(c, "missing auth token")
	}

	_, claims, err := utils.ParseToken(tokenString)
	if err != nil {
		return utils.Unauthorized(c, "invalid token")
	}
	c.Locals("claims", claims)
	// Also set userID for convenience
	c.Locals("userID", claims.UserID)

	return c.Next()
}

// AdminAuthMiddleware verifies that the request has valid admin claims.
func AdminAuthMiddleware(c *fiber.Ctx) error {
	claims, ok := c.Locals("claims").(*models.UserClaims)
	if !ok {
		log.Println("Claims not found in context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid claims"})
	}

	if !claims.HasPermission(models.PermissionReadAdmin) {
		log.Println("Insufficient permissions")
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Insufficient permissions"})
	}

	log.Println("Admin authenticated successfully")
	return c.Next()
}

// HasPermission returns a middleware that checks for a specific permission.
func HasPermission(permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals("claims").(*models.UserClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}
		log.Printf("Checking permission: %s", permission)
		log.Printf("User claims: %+v", claims)
		if claims.HasPermission(permission) {
			return c.Next()
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Insufficient permissions"})
	}
}

// Protected is a sample middleware for routes requiring a minimum role.
func Protected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, ok := c.Locals("claims").(*models.UserClaims)
		if !ok || claims == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}

		requiredRole := getRequiredRole(c.Path())
		if !hasRequiredRole(claims.Role, requiredRole) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Insufficient permissions"})
		}

		return c.Next()
	}
}

// getRequiredRole defines the minimum role required based on the request path.
func getRequiredRole(path string) string {
	switch {
	case strings.HasPrefix(path, "/api/merchant"):
		return "merchant"
	case strings.HasPrefix(path, "/api/enterprise"):
		return "enterprise"
	case strings.HasPrefix(path, "/api/admin"):
		return "admin"
	case strings.HasPrefix(path, "/api/wallet"):
		return "user" // Example: allow both users and merchants for wallet endpoints.
	default:
		return "user"
	}
}

// hasRequiredRole compares the user role and the required role based on a hierarchy.
func hasRequiredRole(userRole, requiredRole string) bool {
	roleHierarchy := map[string]int{
		"user":       1,
		"merchant":   2,
		"enterprise": 3,
		"admin":      4,
	}

	userRoleLevel := roleHierarchy[userRole]
	requiredRoleLevel := roleHierarchy[requiredRole]

	// Optionally allow merchants to access user endpoints.
	if userRole == "merchant" && requiredRole == "user" {
		return true
	}

	return userRoleLevel >= requiredRoleLevel
}
