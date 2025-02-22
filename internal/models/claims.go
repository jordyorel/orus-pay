package models

import "github.com/golang-jwt/jwt/v5"

// UserClaims represents the claims in a JWT token
type UserClaims struct {
	jwt.RegisteredClaims
	UserID       uint     `json:"user_id"`
	Email        string   `json:"email"`
	Role         string   `json:"role"`
	Permissions  []string `json:"permissions"`
	TokenVersion int      `json:"token_version"`
}

// HasPermission checks if the claims include a specific permission
func (c *UserClaims) HasPermission(permission string) bool {
	for _, p := range c.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}
