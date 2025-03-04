package handlers

import (
	qr "orus/internal/services/qr_code"
	"orus/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

type QRHandler struct {
	qrService qr.Service
}

func NewQRHandler(qrService qr.Service) *QRHandler {
	return &QRHandler{
		qrService: qrService,
	}
}

// GenerateQR generates a QR code for a user
func (h *QRHandler) GenerateQR(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	qrCode, err := h.qrService.GetUserReceiveQR(c.Context(), userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to generate QR code")
	}

	return response.Success(c, "QR code generated", qrCode)
}

// GetPaymentQR gets a payment QR code for a user
func (h *QRHandler) GetPaymentQR(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	qrCode, err := h.qrService.GetUserPaymentCodeQR(c.Context(), userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to get payment QR code")
	}

	return response.Success(c, "Payment QR code retrieved", qrCode)
}

// GetUserQRCodes gets all QR codes for a user
func (h *QRHandler) GetUserQRCodes(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	qrCodes, err := h.qrService.GetUserQRCodes(c.Context(), userID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to get QR codes")
	}

	return response.Success(c, "QR codes retrieved", qrCodes)
}
