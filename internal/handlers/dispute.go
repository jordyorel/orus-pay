package handlers

import (
	"orus/internal/models"
	"orus/internal/services/dispute"
	"orus/internal/utils/response"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type DisputeHandler struct {
	disputeService *dispute.Service
}

func NewDisputeHandler(disputeService *dispute.Service) *DisputeHandler {
	return &DisputeHandler{disputeService: disputeService}
}

func (h *DisputeHandler) FileDispute(c *fiber.Ctx) error {
	var input struct {
		TransactionID uint   `json:"transaction_id"`
		Reason        string `json:"reason"`
	}

	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request format")
	}

	claims := c.Locals("claims").(*models.UserClaims)
	dispute, err := h.disputeService.FileDispute(input.TransactionID, claims.UserID, input.Reason)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return response.Success(c, "Dispute filed successfully", dispute)
}

func (h *DisputeHandler) GetDisputes(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)
	disputes, err := h.disputeService.GetDisputes(claims.UserID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return response.Success(c, "Disputes retrieved successfully", disputes)
}

func (h *DisputeHandler) GetMerchantDisputes(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)
	disputes, err := h.disputeService.GetMerchantDisputes(claims.UserID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return response.Success(c, "Merchant disputes retrieved successfully", disputes)
}

func (h *DisputeHandler) RefundDispute(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)

	// Check if the user is a merchant
	if claims.Role != "merchant" {
		return response.Error(c, fiber.StatusForbidden, "You do not have permission to access this endpoint")
	}

	disputeIDStr := c.Params("id")                            // Get the dispute ID as a string
	disputeID, err := strconv.ParseUint(disputeIDStr, 10, 32) // Convert string to uint
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid dispute ID")
	}

	err = h.disputeService.ProcessRefund(uint(disputeID)) // Pass the converted uint
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return response.Success(c, "Refund processed successfully", nil)
}
