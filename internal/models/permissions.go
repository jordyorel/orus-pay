package models

// Permission constants
const (
	// Wallet permissions
	PermissionWalletRead  = "wallet:read"
	PermissionWalletWrite = "wallet:write"

	// Transaction permissions
	PermissionTransactionRead  = "transaction:read"
	PermissionTransactionWrite = "transaction:write"

	// Credit card permissions
	PermissionCreditCardWrite = "creditcard:write"

	// User permissions
	PermissionChangePassword = "user:change-password"

	// Merchant permissions
	PermissionMerchantCreate      = "merchant:create"
	PermissionMerchantRead        = "merchant:read"
	PermissionMerchantWrite       = "merchant:write"
	PermissionMerchantTransaction = "merchant:transaction"

	// Payment permissions
	PermissionPaymentWrite = "payment:write"

	// Admin permissions
	PermissionReadAdmin  = "admin:read"
	PermissionWriteAdmin = "admin:write"

	// User management permissions
	PermissionUserRead  = "user:read"
	PermissionUserWrite = "user:write"
)

// GetDefaultPermissions returns default permissions based on role
func GetDefaultPermissions(role string) []string {
	switch role {
	case "admin":
		return []string{
			PermissionReadAdmin,
			PermissionWriteAdmin,
			PermissionWalletRead,
			PermissionWalletWrite,
			PermissionTransactionRead,
			PermissionTransactionWrite,
			PermissionCreditCardWrite,
			PermissionUserRead,
			PermissionUserWrite,
			PermissionChangePassword,
			PermissionMerchantRead,
			PermissionMerchantWrite,
			PermissionMerchantCreate,
			PermissionPaymentWrite,
		}
	case "regular", "user":
		return []string{
			PermissionWalletRead,
			PermissionWalletWrite,
			PermissionTransactionRead,
			PermissionTransactionWrite,
			PermissionCreditCardWrite,
			PermissionChangePassword,
			PermissionPaymentWrite,
			PermissionMerchantCreate,
		}
	case "merchant":
		return []string{
			PermissionWalletRead,
			PermissionWalletWrite,
			PermissionTransactionRead,
			PermissionTransactionWrite,
			PermissionCreditCardWrite,
			PermissionChangePassword,
			PermissionMerchantCreate,
			PermissionMerchantRead,
			PermissionMerchantWrite,
			PermissionMerchantTransaction,
		}
	default:
		return []string{}
	}
}
