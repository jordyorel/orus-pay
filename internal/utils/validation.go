package utils

import "strings"

func HasSpecialChar(s string) bool {
	specialChars := "!@#$%^&*()_+-=[]{}|;:,.<>?`~"
	for _, char := range s {
		if strings.ContainsRune(specialChars, char) {
			return true
		}
	}
	return false
}
