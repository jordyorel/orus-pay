package validation

import "regexp"

// HasSpecialChar checks if a string contains at least one special character
func HasSpecialChar(s string) bool {
	specialChars := regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`)
	return specialChars.MatchString(s)
}
