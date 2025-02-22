package utils

import (
	"crypto/rand"
	"encoding/base64"
)

func GenerateSecureCode() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
