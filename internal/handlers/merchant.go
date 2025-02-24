package handlers

import (
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/services/merchant"
	qr "orus/internal/services/qr_code"

	"orus/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

type MerchantHandler struct {
	merchantService *merchant.Service
	qrService       qr.Service
}

func NewMerchantHandler(merchantSvc *merchant.Service, qrSvc qr.Service) *MerchantHandler {
	return &MerchantHandler{
		merchantService: merchantSvc,
		qrService:       qrSvc,
	}
}

func (h *MerchantHandler) CreateMerchant(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	var merchant models.Merchant
	if err := c.BodyParser(&merchant); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request format",
		})
	}

	merchant.UserID = userID

	// Create merchant profile
	if err := h.merchantService.CreateMerchant(&merchant); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return response.Success(c, "Merchant profile created successfully", fiber.Map{
		"merchant": merchant,
	})
}

func (h *MerchantHandler) GetMerchantProfile(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)
	merchant, err := repositories.GetMerchantByUserID(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Merchant profile not found",
		})
	}
	return c.JSON(merchant)
}

func (h *MerchantHandler) ProcessDirectCharge(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)
	var input merchant.ChargeInput
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request format")
	}

	tx, err := h.merchantService.ProcessDirectCharge(claims.UserID, input)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return response.Success(c, "Transaction processed successfully", tx)
}

func (h *MerchantHandler) UpdateMerchantProfile(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)
	var input merchant.UpdateMerchantInput

	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.merchantService.UpdateMerchantProfile(claims.UserID, input); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to update merchant profile")
	}

	return response.Success(c, "Profile updated successfully", nil)
}

func (h *MerchantHandler) GenerateAPIKey(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)

	apiKey, err := h.merchantService.GenerateAPIKey(claims.UserID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to generate API key")
	}

	return response.Success(c, "API key generated", fiber.Map{"api_key": apiKey})
}

func (h *MerchantHandler) SetWebhookURL(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)
	var input struct {
		WebhookURL string `json:"webhook_url"`
	}

	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.merchantService.SetWebhookURL(claims.UserID, input.WebhookURL); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to set webhook URL")
	}

	return response.Success(c, "Webhook URL updated successfully", nil)
}
