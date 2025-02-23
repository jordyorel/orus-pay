package utils

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateUniqueID creates a secure random string of specified length
func GenerateUniqueID(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
