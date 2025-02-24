package validation

import (
	"context"
	"orus/internal/domain/qr"
	"orus/internal/errors"
	"orus/internal/models"
)

func ValidateQRRequest(req qr.GenerateQRRequest) error {
	if req.UserID == 0 {
		return &errors.DomainError{
			Code:    "INVALID_REQUEST",
			Message: "user ID is required",
		}
	}

	switch req.QRType {
	case qr.TypeStatic, qr.TypeReceive, qr.TypePaymentCode, qr.TypeDynamic, qr.TypePayment:
		// Valid types
	default:
		return &errors.DomainError{
			Code:    "INVALID_REQUEST",
			Message: "invalid QR type",
		}
	}

	// Allow -1 for unlimited uses, or positive numbers
	if req.MaxUses < -1 {
		return &errors.DomainError{
			Code:    "INVALID_REQUEST",
			Message: "max uses must be -1 (unlimited) or a positive number",
		}
	}

	if req.QRType == qr.TypeDynamic && (req.Amount == nil || *req.Amount <= 0) {
		return &errors.DomainError{
			Code:    "INVALID_REQUEST",
			Message: "amount is required for dynamic QR",
		}
	}

	return nil
}

func ValidateQRPayment(qr *models.QRCode, amount float64) error {
	if qr.Amount != nil && *qr.Amount != amount {
		return &errors.DomainError{
			Code:    "INVALID_AMOUNT",
			Message: "amount mismatch",
		}
	}
	return nil
}

func CheckQRLimits(ctx context.Context, qr *models.QRCode, amount float64) error {
	if qr.DailyLimit != nil && amount > *qr.DailyLimit {
		return &errors.DomainError{
			Code:    "LIMIT_EXCEEDED",
			Message: "daily limit exceeded",
		}
	}
	return nil
}
