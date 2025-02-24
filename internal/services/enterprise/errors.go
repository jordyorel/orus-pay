package enterprise

import "errors"

var (
	ErrInvalidContract = errors.New("invalid contract dates")
	ErrInvalidPricing  = errors.New("invalid pricing configuration")
	ErrInvalidAPIKey   = errors.New("invalid API key configuration")
)
