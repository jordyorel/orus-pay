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
	KeyID    KeyType = "id"
	KeyEmail KeyType = "email"
	KeyPhone KeyType = "phone"
	KeyCode  KeyType = "code"
)

// GenerateKey creates a standardized cache key
func GenerateKey(entity EntityType, keyType KeyType, value interface{}) string {
	return fmt.Sprintf("%s:%s:%v", entity, keyType, value)
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
