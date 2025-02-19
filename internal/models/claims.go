package models

import "github.com/golang-jwt/jwt/v5"

// Application permissions
const (
	// Admin permissions
	PermissionReadAdmin  = "admin:read"
	PermissionWriteAdmin = "admin:write"

	// User permissions
	PermissionWalletRead       = "wallet:read"
	PermissionWalletWrite      = "wallet:write"
	PermissionTransactionRead  = "transaction:read"
	PermissionTransactionWrite = "transaction:write"
	PermissionCreditCardWrite  = "creditcard:write"
	PermissionChangePassword   = "user:change-password"

	// Merchant permissions
	PermissionMerchantCreate      = "merchant:create"
	PermissionMerchantRead        = "merchant:read"
	PermissionMerchantWrite       = "merchant:write"
	PermissionMerchantTransaction = "merchant:transaction"

	// Payment permissions
	PermissionPaymentWrite = "payment:write"

	// User management permissions
	PermissionUserRead  = "user:read"
	PermissionUserWrite = "user:write"
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
			PermissionMerchantRead,
			PermissionMerchantWrite,
		}
	case "merchant":
		return []string{
			PermissionWalletRead,
			PermissionWalletWrite,
			PermissionTransactionRead,
			PermissionTransactionWrite,
			PermissionCreditCardWrite,
			PermissionChangePassword,
			PermissionMerchantRead,
			PermissionMerchantWrite,
			PermissionMerchantTransaction,
			PermissionMerchantCreate,
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
