package transaction

import "errors"

// Service errors
var (
	ErrTransactionNotFound  = errors.New("transaction not found")
	ErrDailyLimitExceeded   = errors.New("daily transaction limit exceeded")
	ErrProcessingFailed     = errors.New("transaction processing failed")
	ErrInvalidStatus        = errors.New("invalid transaction status")
	ErrDuplicateTransaction = errors.New("duplicate transaction")
	ErrProcessingTimeout    = errors.New("transaction processing timeout")
	ErrInvalidCallback      = errors.New("invalid callback URL")
)
