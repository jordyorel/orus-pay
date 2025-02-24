package merchant

import "errors"

var (
	ErrMerchantInactive = errors.New("merchant is not active")
	ErrInvalidAmount    = errors.New("invalid transaction amount")
	ErrLimitExceeded    = errors.New("transaction limit exceeded")
)
