package handlers

import (
	"context"
	"errors"
	"fmt"
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/services/wallet"
	"orus/internal/utils"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type WalletHandler struct {
	walletService wallet.Service
}

func NewWalletHandler(walletService wallet.Service) *WalletHandler {
	return &WalletHandler{
		walletService: walletService,
	}
}

// extractUserClaims is a helper function to reduce duplication
func extractUserClaims(c *fiber.Ctx) (*models.UserClaims, error) {
	claims, ok := c.Locals("claims").(*models.UserClaims)
	if !ok || claims == nil {
		return nil, fiber.ErrUnauthorized
	}
	return claims, nil
}

func (h *WalletHandler) GetWallet(c *fiber.Ctx) error {
	claims, err := extractUserClaims(c)
	if err != nil {
		return utils.Unauthorized(c, "invalid claims")
	}

	wallet, err := h.walletService.GetWallet(c.Context(), claims.UserID)
	if err != nil {
		return utils.InternalError(c, "Failed to get wallet")
	}

	return utils.Success(c, fiber.Map{
		"wallet": wallet,
	})
}

func (h *WalletHandler) TopUpWallet(c *fiber.Ctx) error {
	claims, err := extractUserClaims(c)
	if err != nil {
		return utils.Unauthorized(c, "invalid claims")
	}

	// Debug log
	fmt.Printf("User Role: %s\n", claims.Role)

	var input struct {
		Amount float64 `json:"amount" validate:"required,gt=0"`
		CardID uint    `json:"card_id" validate:"required"`
	}

	if err := c.BodyParser(&input); err != nil {
		return utils.BadRequest(c, "Invalid request format")
	}

	if input.Amount <= 0 {
		return utils.BadRequest(c, "Amount must be greater than 0")
	}

	// Create context with user role
	ctx := context.WithValue(c.Context(), wallet.UserRoleContextKey, claims.Role)

	err = h.walletService.TopUp(ctx, claims.UserID, input.CardID, input.Amount)
	if err != nil {
		return utils.InternalError(c, err.Error())
	}

	return utils.Success(c, fiber.Map{
		"message": "Top up successful",
		"amount":  input.Amount,
	})
}

func (h *WalletHandler) WithdrawToCard(c *fiber.Ctx) error {
	claims, err := extractUserClaims(c)
	if err != nil {
		return utils.Unauthorized(c, "invalid claims")
	}

	var input struct {
		Amount float64 `json:"amount" validate:"required,gt=0"`
		CardID uint    `json:"card_id" validate:"required"`
	}

	if err := c.BodyParser(&input); err != nil {
		return utils.BadRequest(c, "Invalid request format")
	}

	if input.Amount <= 0 {
		return utils.BadRequest(c, "Amount must be greater than 0")
	}

	// Get fee percentage from service
	feePercent := h.walletService.GetWithdrawalFeePercent()
	fee := input.Amount * feePercent

	err = h.walletService.Withdraw(c.Context(), claims.UserID, input.CardID, input.Amount)
	if err != nil {
		if errors.Is(err, repositories.ErrCardNotFound) {
			return utils.BadRequest(c, "Card not found")
		}
		if strings.Contains(err.Error(), "invalid card") {
			return utils.BadRequest(c, "Invalid card or access denied")
		}
		if strings.Contains(err.Error(), "not active") {
			return utils.BadRequest(c, "Card is not active")
		}
		return utils.InternalError(c, err.Error())
	}

	// Get updated wallet balance
	wallet, err := h.walletService.GetWallet(c.Context(), claims.UserID)
	if err != nil {
		return utils.InternalError(c, "Failed to get updated wallet balance")
	}

	return utils.Success(c, fiber.Map{
		"message":        "Withdrawal successful",
		"amount":         input.Amount,
		"fee":            fee,
		"total_deducted": input.Amount + fee,
		"new_balance":    wallet.Balance,
	})
}
