package handlers

import (
	"context"
	"fmt"
	"orus/internal/models"
	"orus/internal/services/payment"
	qr "orus/internal/services/qr_code"
	"orus/internal/services/wallet"
	"orus/internal/utils"
	"orus/internal/utils/response"
	"orus/internal/validation"

	"github.com/gofiber/fiber/v2"
)

type PaymentHandler struct {
	qrService      qr.Service
	paymentService payment.Service
}

func NewPaymentHandler(qrSvc qr.Service, paymentSvc payment.Service) *PaymentHandler {
	return &PaymentHandler{
		qrService:      qrSvc,
		paymentService: paymentSvc,
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

	var input struct {
		ReceiverID  uint    `json:"receiver_id"`
		Amount      float64 `json:"amount"`
		Description string  `json:"description"`
	}

	if err := c.BodyParser(&input); err != nil {
		return utils.BadRequest(c, "Invalid request format")
	}

	// Create context with user role
	ctx := context.WithValue(c.Context(), wallet.UserRoleContextKey, claims.Role)

	// Debug log
	fmt.Printf("SendMoney - User Role: %s, From: %d, To: %d, Amount: %.2f\n",
		claims.Role, claims.UserID, input.ReceiverID, input.Amount)

	tx, err := h.paymentService.SendMoney(
		ctx,
		claims.UserID,
		input.ReceiverID,
		input.Amount,
		input.Description,
	)
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

	// Create context with user role
	ctx := context.WithValue(c.Context(), wallet.UserRoleContextKey, claims.Role)

	// Process payment based on type
	var result *models.Transaction
	var err error

	switch req.PaymentType {
	case "merchant_payment":
		result, err = h.paymentService.ProcessMerchantPayment(
			ctx,
			claims.UserID,
			req.RecipientID,
			req.Amount,
			req.Description,
		)
	default:
		result, err = h.paymentService.SendMoney(
			ctx,
			claims.UserID,
			req.RecipientID,
			req.Amount,
			req.Description,
		)
	}

	if err != nil {
		return response.ServerError(c, err.Error())
	}

	return response.Success(c, "Payment processed successfully", result)
}

// Supports multiple payment flows:
// - QR code payments
// - P2P transfers
// - Merchant payments
// - Cross-role transactions (user ↔ merchant)
