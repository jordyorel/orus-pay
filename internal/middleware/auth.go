// Package middleware provides HTTP middleware components for the application.
// It includes authentication, authorization, and other request processing middleware
// that can be used with the fiber web framework.
package middleware

import (
	"log"
	"strings"

	"orus/internal/config"
	"orus/internal/models"
	"orus/internal/services/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware handles JWT token validation and user authentication.
// It extracts the JWT token from the Authorization header, validates it,
// and adds the user claims to the request context.
type AuthMiddleware struct {
	authService auth.Service
}

func NewAuthMiddleware(authService auth.Service) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// Handler validates JWT tokens and adds claims to the request context.
// It checks for:
// - Presence of Authorization header with Bearer token
// - Valid JWT signature
// - Token expiration
// - Token version matches current user version
func (m *AuthMiddleware) Handler(c *fiber.Ctx) error {
	// Get the Authorization header
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		log.Println("Missing Authorization header")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization header"})
	}

	// Check if the header has the Bearer prefix
	if !strings.HasPrefix(authHeader, "Bearer ") {
		log.Println("Invalid Authorization format")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization format"})
	}

	// Extract the token
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// Parse and validate the token
	token, err := jwt.ParseWithClaims(tokenString, &models.UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.GetEnv("JWT_SECRET", "your-secret-key")), nil
	})

	if err != nil {
		log.Printf("Token validation error: %v", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
	}

	// Check if the token is valid
	if !token.Valid {
		log.Println("Token is invalid")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
	}

	// Extract the claims
	claims, ok := token.Claims.(*models.UserClaims)
	if !ok {
		log.Println("Failed to extract claims")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid claims"})
	}

	// Get current token version from auth service
	currentVersion, err := m.authService.GetUserTokenVersion(claims.UserID)
	if err != nil {
		log.Printf("Error getting token version: %v", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
	}

	// Check if token version matches current version
	if claims.TokenVersion != currentVersion {
		log.Printf("Token version mismatch. Token: %d, Current: %d", claims.TokenVersion, currentVersion)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "token has been invalidated"})
	}

	// Store the claims in the context
	c.Locals("claims", claims)
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

	// Add debug logging
	log.Printf("User role: %s", claims.Role)
	log.Printf("User permissions: %v", claims.Permissions)

	// Check both role and permission
	if claims.Role != "admin" || !claims.HasPermission(models.PermissionReadAdmin) {
		log.Println("Insufficient permissions")
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Insufficient permissions"})
	}

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

		// If user is admin, allow all permissions
		if claims.Role == "admin" {
			return c.Next()
		}

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

// Create a middleware factory that accepts the JWT secret
func CreateAuthMiddleware(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get the Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			log.Println("Missing Authorization header")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization header"})
		}

		// Check if the header has the Bearer prefix
		if !strings.HasPrefix(authHeader, "Bearer ") {
			log.Println("Invalid Authorization format")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization format"})
		}

		// Extract the token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Parse and validate the token
		token, err := jwt.ParseWithClaims(tokenString, &models.UserClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		if err != nil {
			log.Printf("Token validation error: %v", err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}

		// Check if the token is valid
		if !token.Valid {
			log.Println("Token is invalid")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}

		// Extract the claims
		claims, ok := token.Claims.(*models.UserClaims)
		if !ok {
			log.Println("Failed to extract claims")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid claims"})
		}

		// Store the claims in the context
		c.Locals("claims", claims)
		c.Locals("userID", claims.UserID)

		return c.Next()
	}
}
