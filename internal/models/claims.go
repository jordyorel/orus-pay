package models

import "github.com/golang-jwt/jwt/v5"

// Application permissions
const (
	// Admin permissions
	PermissionReadAdmin  = "admin:read"
	PermissionWriteAdmin = "admin:write"

	// Wallet permissions
	PermissionWalletRead  = "wallet:read"
	PermissionWalletWrite = "wallet:write"

	// Transaction permissions
	PermissionTransactionRead  = "transaction:read"
	PermissionTransactionWrite = "transaction:write"

	// Payment permissions
	PermissionPaymentWrite = "payment:write"

	// Credit card permissions
	PermissionCreditCardWrite = "creditcard:write"

	// User management permissions
	PermissionUserRead       = "user:read"
	PermissionUserWrite      = "user:write"
	PermissionChangePassword = "user:change_password"
)

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

// GetDefaultPermissions returns default permissions based on role
func GetDefaultPermissions(role string) []string {
	switch role {
	case "admin":
		return []string{
			PermissionWalletRead,
			PermissionWalletWrite,
			PermissionTransactionRead,
			PermissionTransactionWrite,
			PermissionPaymentWrite,
			PermissionCreditCardWrite,
			PermissionUserRead,
			PermissionUserWrite,
			PermissionChangePassword,
			PermissionReadAdmin,
			PermissionWriteAdmin,
		}
	case "user":
		return []string{
			PermissionWalletRead,
			PermissionWalletWrite,
			PermissionTransactionRead,
			PermissionTransactionWrite,
			PermissionPaymentWrite,
			PermissionCreditCardWrite,
			PermissionUserRead,
			PermissionChangePassword,
		}
	default:
		return []string{}
	}
}
