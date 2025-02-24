package errors

var (
	ErrInsufficientBalance = &DomainError{
		Code:    "INSUFFICIENT_BALANCE",
		Message: "insufficient wallet balance",
	}
	ErrInvalidAmount = &DomainError{
		Code:    "INVALID_AMOUNT",
		Message: "invalid amount",
	}
	ErrWalletNotFound = &DomainError{
		Code:    "WALLET_NOT_FOUND",
		Message: "wallet not found",
	}
)
