package handlers

import (
	"context"
	"orus/internal/models"
	"orus/internal/services/transfer"
	"orus/internal/services/wallet"
	"orus/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// TransferHandler exposes P2P transfer endpoints.
type TransferHandler struct {
	service transfer.Service
}

// NewTransferHandler creates a new TransferHandler.
func NewTransferHandler(s transfer.Service) *TransferHandler { return &TransferHandler{service: s} }

// Transfer handles POST /transfer requests.
func (h *TransferHandler) Transfer(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)

	var req struct {
		ReceiverID  uint    `json:"receiver_id"`
		Amount      float64 `json:"amount"`
		Description string  `json:"description"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request")
	}

	ctx := context.WithValue(c.Context(), wallet.UserRoleContextKey, claims.Role)
	tx, err := h.service.Transfer(ctx, claims.UserID, req.ReceiverID, req.Amount, req.Description)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error())
	}
	return response.Success(c, "transfer completed", tx)
}
