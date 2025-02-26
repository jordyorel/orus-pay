package handlers

import (
	"context"
	"fmt"
	"orus/internal/models"
	qr "orus/internal/services/qr_code"
	transaction "orus/internal/services/transaction"
	"orus/internal/services/wallet"
	"orus/internal/utils"
	"orus/internal/utils/response"
	"orus/internal/validation"

	"github.com/gofiber/fiber/v2"
)

type PaymentHandler struct {
	qrService          qr.Service
	transactionService transaction.Service
}

func NewPaymentHandler(qrSvc qr.Service, txSvc transaction.Service) *PaymentHandler {
	return &PaymentHandler{
		qrService:          qrSvc,
		transactionService: txSvc,
	}
}

// ProcessQRPayment handles QR code payments for both users and merchants
func (h *PaymentHandler) ProcessQRPayment(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)

	var input models.QRPaymentRequest

	if err := c.BodyParser(&input); err != nil {
		return utils.BadRequest(c, "Invalid request format")
	}

	v := validation.New()
	v.QRPayment(&input)
	if !v.Valid() {
		for _, msg := range v.Errors {
			return utils.BadRequest(c, msg)
		}
	}

	// Enrich metadata based on who is scanning
	if input.Metadata == nil {
		input.Metadata = make(map[string]any)
	}
	input.Metadata["scanner_role"] = claims.Role
	input.Metadata["scanner_id"] = claims.UserID

	// Add payment context to description
	if claims.Role == "merchant" {
		input.Description = fmt.Sprintf("Merchant payment: %s", input.Description)
	} else {
		input.Description = fmt.Sprintf("User payment: %s", input.Description)
	}

	tx, err := h.qrService.ProcessQRPayment(
		c.Context(),
		input.QRCode,
		input.Amount,
		claims.UserID,
		input.Description,
		input.Metadata,
	)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error())
	}

	return response.Success(c, "Payment successful", tx)
}

// SendMoney handles direct P2P transfers
func (h *PaymentHandler) SendMoney(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)

	var input transaction.TransferRequest

	if err := c.BodyParser(&input); err != nil {
		return utils.BadRequest(c, "Invalid request format")
	}

	// Set sender ID before validation
	input.SenderID = claims.UserID

	// Create context with user role
	ctx := context.WithValue(c.Context(), wallet.UserRoleContextKey, claims.Role)

	// Debug log
	fmt.Printf("SendMoney - User Role: %s, From: %d, To: %d, Amount: %.2f\n",
		claims.Role, claims.UserID, input.ReceiverID, input.Amount)

	v := validation.New()
	v.Transfer(&input)
	if !v.Valid() {
		for _, msg := range v.Errors {
			return utils.BadRequest(c, fmt.Sprintf("Validation error: %s", msg))
		}
	}

	tx, err := h.transactionService.ProcessP2PTransfer(ctx, input)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error())
	}

	return response.Success(c, "Transfer successful", tx)
}

func (h *PaymentHandler) ProcessPayment(c *fiber.Ctx) error {
	var req models.PaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return response.ValidationError(c, "Invalid request format")
	}

	v := validation.New()
	v.Payment(&req) // Use the Payment validation method

	if !v.Valid() {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"errors": v.Errors,
		})
	}

	// Get user ID from context
	claims := c.Locals("claims").(*models.UserClaims)
	if claims == nil {
		return response.Unauthorized(c)
	}

	// Create transaction request
	tx := &models.Transaction{
		Type:        req.PaymentType,
		SenderID:    claims.UserID,
		ReceiverID:  req.RecipientID,
		Amount:      req.Amount,
		Description: req.Description,
		Status:      "pending",
	}

	// Process transaction
	result, err := h.transactionService.ProcessTransaction(c.Context(), tx)
	if err != nil {
		return response.ServerError(c, err.Error())
	}

	return response.Success(c, "Payment processed successfully", result)
}
