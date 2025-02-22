package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func GenerateAPIKey() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("mk_%s", hex.EncodeToString(bytes))
}
