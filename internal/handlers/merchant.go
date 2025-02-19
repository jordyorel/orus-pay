package handlers

import (
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/services"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type MerchantHandler struct {
	merchantService *services.MerchantService
}

func NewMerchantHandler() *MerchantHandler {
	return &MerchantHandler{
		merchantService: services.NewMerchantService(),
	}
}

func (h *MerchantHandler) CreateMerchant(c *fiber.Ctx) error {
	var input struct {
		UserID             uint   `json:"user_id"`
		BusinessName       string `json:"business_name"`
		BusinessType       string `json:"business_type"`
		BusinessAddress    string `json:"business_address"`
		BusinessID         string `json:"business_id"`
		TaxID              string `json:"tax_id"`
		Website            string `json:"website,omitempty"`
		MerchantCategory   string `json:"merchant_category"`
		LegalEntityType    string `json:"legal_entity_type"`
		RegistrationNumber string `json:"registration_number"`
		YearEstablished    int    `json:"year_established,omitempty"`
		SupportEmail       string `json:"support_email"`
		SupportPhone       string `json:"support_phone"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate all required fields
	requiredFields := map[string]string{
		"business_name":       input.BusinessName,
		"business_type":       input.BusinessType,
		"business_address":    input.BusinessAddress,
		"business_id":         input.BusinessID,
		"tax_id":              input.TaxID,
		"merchant_category":   input.MerchantCategory,
		"legal_entity_type":   input.LegalEntityType,
		"registration_number": input.RegistrationNumber,
		"support_email":       input.SupportEmail,
		"support_phone":       input.SupportPhone,
	}

	for field, value := range requiredFields {
		if value == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": field + " is required",
			})
		}
	}

	// Check if user exists and has merchant role
	user, err := repositories.GetUserByID(input.UserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	if user.Role != "merchant" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User must have merchant role",
		})
	}

	// Check if merchant already exists for this user
	_, err = repositories.GetMerchantByUserID(input.UserID)
	if err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Merchant profile already exists for this user",
		})
	}
	if err != gorm.ErrRecordNotFound {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check existing merchant",
		})
	}

	// Create merchant with all fields
	merchant := &models.Merchant{
		UserID:             input.UserID,
		BusinessName:       input.BusinessName,
		BusinessType:       input.BusinessType,
		BusinessAddress:    input.BusinessAddress,
		BusinessID:         input.BusinessID,
		TaxID:              input.TaxID,
		Website:            input.Website,
		MerchantCategory:   input.MerchantCategory,
		LegalEntityType:    input.LegalEntityType,
		RegistrationNumber: input.RegistrationNumber,
		YearEstablished:    input.YearEstablished,
		SupportEmail:       input.SupportEmail,
		SupportPhone:       input.SupportPhone,
		VerificationStatus: "pending_review",
		ProcessingFeeRate:  2.5,
	}

	if err := h.merchantService.CreateMerchant(merchant); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(merchant)
}

func (h *MerchantHandler) ProcessTransaction(c *fiber.Ctx) error {
	var req struct {
		Amount float64 `json:"amount"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	merchantID, err := c.ParamsInt("merchantId")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid merchant ID"})
	}

	if err := h.merchantService.ProcessTransaction(uint(merchantID), req.Amount); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Transaction processed successfully"})
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

func (h *MerchantHandler) UpdateMerchantProfile(c *fiber.Ctx) error {
	var input struct {
		BusinessName    string  `json:"business_name"`
		BusinessType    string  `json:"business_type"`
		BusinessAddress string  `json:"business_address"`
		ProcessingFee   float64 `json:"processing_fee"`
		WebhookURL      string  `json:"webhook_url"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	claims := c.Locals("claims").(*models.UserClaims)
	merchant, err := repositories.GetMerchantByUserID(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Merchant not found",
		})
	}

	// Update fields
	merchant.BusinessName = input.BusinessName
	merchant.BusinessType = input.BusinessType
	merchant.BusinessAddress = input.BusinessAddress
	merchant.ProcessingFeeRate = input.ProcessingFee
	merchant.WebhookURL = input.WebhookURL

	if err := repositories.UpdateMerchant(merchant); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update merchant profile",
		})
	}

	return c.JSON(merchant)
}

func (h *MerchantHandler) GetMerchantTransactions(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)
	transactions, err := repositories.GetMerchantTransactions(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch transactions",
		})
	}
	return c.JSON(fiber.Map{
		"transactions": transactions,
	})
}

func (h *MerchantHandler) GenerateAPIKey(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)
	apiKey, err := repositories.GenerateMerchantAPIKey(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate API key",
		})
	}
	return c.JSON(fiber.Map{
		"api_key": apiKey,
	})
}

func (h *MerchantHandler) SetWebhookURL(c *fiber.Ctx) error {
	var input struct {
		WebhookURL string `json:"webhook_url"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	claims := c.Locals("claims").(*models.UserClaims)
	if err := repositories.SetMerchantWebhookURL(claims.UserID, input.WebhookURL); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to set webhook URL",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Webhook URL updated successfully",
	})
}
