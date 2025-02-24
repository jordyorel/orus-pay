package handlers

import (
	"fmt"
	"orus/internal/models"
	qr "orus/internal/services/qr_code"
	transaction "orus/internal/services/transaction"
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

	var input struct {
		QRCode      string                 `json:"qr_code" validate:"required"`
		Amount      float64                `json:"amount" validate:"required,gt=0"`
		Description string                 `json:"description"`
		Metadata    map[string]interface{} `json:"metadata"`
	}

	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request format")
	}

	// Enrich metadata based on who is scanning
	if input.Metadata == nil {
		input.Metadata = make(map[string]interface{})
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

	var input struct {
		ReceiverID  uint    `json:"receiver_id" validate:"required"`
		Amount      float64 `json:"amount" validate:"required,gt=0"`
		Description string  `json:"description"`
	}

	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request format")
	}

	req := transaction.TransferRequest{
		SenderID:    claims.UserID, // Set from authenticated user
		ReceiverID:  input.ReceiverID,
		Amount:      input.Amount,
		Description: input.Description,
	}

	tx, err := h.transactionService.ProcessP2PTransfer(c.Context(), req)
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
