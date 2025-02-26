package wallet

import "errors"

// Service errors
var (
	// Wallet-specific errors
	ErrInvalidCurrency      = errors.New("invalid currency")
	ErrDailyLimitExceeded   = errors.New("daily limit exceeded")
	ErrMonthlyLimitExceeded = errors.New("monthly limit exceeded")
	ErrWalletLocked         = errors.New("wallet is locked")
	ErrInvalidOperation     = errors.New("invalid operation")
	ErrTransactionFailed    = errors.New("transaction failed")
)
