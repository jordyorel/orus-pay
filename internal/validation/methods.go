package validation

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// Validator defines validation methods
type Validator struct {
	Errors map[string]string
}

// New creates a new validator
func New() *Validator {
	return &Validator{Errors: make(map[string]string)}
}

// Valid checks if there are any validation errors
func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

// AddError adds an error to the validator
func (v *Validator) AddError(field, message string) {
	v.Errors[field] = message
}

// Check adds an error if the condition is false
func (v *Validator) Check(ok bool, field, message string) {
	if !ok {
		v.AddError(field, message)
	}
}

// Email validates email format
func (v *Validator) Email(field, email string) {
	v.Check(emailRegex.MatchString(email), field, "must be a valid email address")
}

// Phone validates phone number format
func (v *Validator) Phone(field, phone string) {
	v.Check(phoneRegex.MatchString(phone), field, "must be a valid phone number")
}

// Required checks if a string is not empty
func (v *Validator) Required(field string, value interface{}) {
	if value == nil {
		v.AddError(field, "must not be nil")
		return
	}

	switch val := value.(type) {
	case string:
		trimmed := strings.TrimSpace(val)
		v.Check(trimmed != "", field, "must not be empty")
	case []string:
		v.Check(len(val) > 0, field, "must contain at least one item")
	case []interface{}:
		v.Check(len(val) > 0, field, "must contain at least one item")
	case float64:
		v.Check(val != 0, field, "must not be zero")
	case int:
		v.Check(val != 0, field, "must not be zero")
	case uint:
		v.Check(val != 0, field, "must not be zero")
	}
}

// MinLength checks if a string has at least n characters
func (v *Validator) MinLength(field string, value string, n int) {
	v.Check(len(value) >= n, field, fmt.Sprintf("must be at least %d characters long", n))
}

// MaxLength checks if a string has at most n characters
func (v *Validator) MaxLength(field string, value string, n int) {
	v.Check(len(value) <= n, field, fmt.Sprintf("must not be more than %d characters long", n))
}

// Range checks if a number is between min and max
func (v *Validator) Range(field string, value float64, min, max float64) {
	v.Check(value >= min && value <= max, field, fmt.Sprintf("must be between %v and %v", min, max))
}

// Future checks if a time is in the future
func (v *Validator) Future(field string, t time.Time) {
	v.Check(t.After(time.Now()), field, "must be in the future")
}

// Password validates password strength
func (v *Validator) Password(field, password string) {
	v.MinLength(field, password, 8)

	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	v.Check(hasUpper, field, "must contain at least one uppercase letter")
	v.Check(hasLower, field, "must contain at least one lowercase letter")
	v.Check(hasNumber, field, "must contain at least one number")
	v.Check(hasSpecial, field, "must contain at least one special character")
}
