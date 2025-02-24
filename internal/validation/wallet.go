package validation

import (
	"context"
	"orus/internal/errors"
	"orus/internal/models"
	"orus/internal/services/wallet"
)

func ValidateWalletOperation(wallet *models.Wallet) error {
	if wallet == nil {
		return &errors.DomainError{
			Code:    "INVALID_WALLET",
			Message: "wallet is nil",
		}
	}

	if wallet.Status != "active" {
		return &errors.DomainError{
			Code:    "INVALID_STATUS",
			Message: "wallet is not active",
		}
	}

	return nil
}

func ValidateTransferRequest(req wallet.TransferRequest) error {
	if req.Amount <= 0 {
		return &errors.DomainError{
			Code:    "INVALID_AMOUNT",
			Message: "amount must be positive",
		}
	}

	if req.FromWalletID == 0 || req.ToWalletID == 0 {
		return &errors.DomainError{
			Code:    "INVALID_REQUEST",
			Message: "invalid wallet IDs",
		}
	}

	return nil
}

func CheckWalletLimits(ctx context.Context, userID uint, amount float64) error {
	// This would typically check against configured limits
	// For now, return nil as the actual implementation would depend on your requirements
	return nil
}
