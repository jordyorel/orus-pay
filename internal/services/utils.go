package services

import (
	"github.com/google/uuid"
)

// GenerateUniqueID generates a unique ID (e.g., UUID)
func GenerateUniqueID() string {
	return uuid.NewString() // Generates a random UUID
}
