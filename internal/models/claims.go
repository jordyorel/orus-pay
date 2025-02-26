package models

import (
	"log"

	"slices"

	"github.com/golang-jwt/jwt/v5"
)

// UserClaims represents the claims in a JWT token
type UserClaims struct {
	jwt.RegisteredClaims
	UserID       uint     `json:"user_id"`
	Email        string   `json:"email"`
	Role         string   `json:"role"`
	TokenType    string   `json:"token_type"`
	Permissions  []string `json:"permissions"`
	TokenVersion int      `json:"token_version"`
}

// HasPermission checks if the user has a specific permission
func (c *UserClaims) HasPermission(permission string) bool {
	// Log permissions for debugging
	log.Printf("User permissions: %v, checking for: %s", c.Permissions, permission)

	// First check explicit permissions in the claims
	if slices.Contains(c.Permissions, permission) {
		return true
	}

	// If no explicit permissions match, check role-based permissions
	for _, p := range GetDefaultPermissions(c.Role) {
		if p == permission {
			return true
		}
	}

	return false
}
