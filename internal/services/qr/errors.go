package qr

import "errors"

// Service errors
var (
	ErrInvalidRequest    = errors.New("invalid request")
	ErrInvalidQRType     = errors.New("invalid QR code type")
	ErrInvalidUserType   = errors.New("invalid user type")
	ErrQRExpired         = errors.New("QR code has expired")
	ErrQRInactive        = errors.New("QR code is not active")
	ErrQRLimitExceeded   = errors.New("QR code usage limit exceeded")
	ErrInvalidAmount     = errors.New("invalid amount")
	ErrInsufficientFunds = errors.New("insufficient funds")
)
