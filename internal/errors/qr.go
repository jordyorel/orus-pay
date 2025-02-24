package errors

var (
	ErrQRInactive = &DomainError{
		Code:    "QR_INACTIVE",
		Message: "QR code is not active",
	}
	ErrQRExpired = &DomainError{
		Code:    "QR_EXPIRED",
		Message: "QR code has expired",
	}
	ErrInvalidQR = &DomainError{
		Code:    "INVALID_QR",
		Message: "invalid QR code",
	}
	ErrQRLimitExceeded = &DomainError{
		Code:    "QR_LIMIT_EXCEEDED",
		Message: "QR code usage limit exceeded",
	}
)
