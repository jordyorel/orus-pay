package cache

import (
	"fmt"
	"strings"
)

type EntityType string

const (
	EntityUser        EntityType = "user"
	EntityWallet      EntityType = "wallet"
	EntityMerchant    EntityType = "merchant"
	EntityQRCode      EntityType = "qrcode"
	EntityTransaction EntityType = "transaction"
)

type KeyType string

const (
	KeyID      KeyType = "id"
	KeyEmail   KeyType = "email"
	KeyPhone   KeyType = "phone"
	KeyCode    KeyType = "code"
	KeyBalance KeyType = "balance"
)

// GenerateKey creates a standardized cache key
func GenerateKey(entity EntityType, keyType KeyType, value interface{}) string {
	return fmt.Sprintf("%s:%s:%v", entity, keyType, value)
}

// GenerateWalletKey creates a cache key for a wallet
func GenerateWalletKey(walletID uint) string {
	return GenerateKey(EntityWallet, KeyID, walletID)
}

// GenerateWalletBalanceKey creates a cache key specifically for a wallet's balance
func GenerateWalletBalanceKey(walletID uint) string {
	return GenerateKey(EntityWallet, KeyBalance, walletID)
}

// GenerateUserWalletKey creates a cache key for a user's wallet
func GenerateUserWalletKey(userID uint) string {
	return fmt.Sprintf("%s:%s:%d:%s", EntityUser, KeyID, userID, EntityWallet)
}

// GenerateCompositeKey creates a cache key with multiple components
func GenerateCompositeKey(entity EntityType, components map[string]interface{}) string {
	var parts []string
	parts = append(parts, string(entity))

	for k, v := range components {
		parts = append(parts, fmt.Sprintf("%s:%v", k, v))
	}

	return strings.Join(parts, ":")
}

// ParseKey extracts components from a cache key
func ParseKey(key string) map[string]string {
	parts := strings.Split(key, ":")
	if len(parts) < 2 {
		return nil
	}

	result := make(map[string]string)
	result["entity"] = parts[0]

	for i := 1; i < len(parts)-1; i += 2 {
		if i+1 < len(parts) {
			result[parts[i]] = parts[i+1]
		}
	}

	return result
}

// InvalidateWalletCache invalidates all cache entries for a wallet
func InvalidateWalletCache(walletID uint) []string {
	keys := []string{
		GenerateWalletKey(walletID),
		GenerateWalletBalanceKey(walletID),
	}
	return keys
}

// InvalidateUserWalletCache invalidates all cache entries for a user's wallet
func InvalidateUserWalletCache(userID uint) []string {
	return []string{
		GenerateUserWalletKey(userID),
	}
}
