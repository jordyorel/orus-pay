package handlers

import (
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/services"

	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
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
	userID := c.Locals("userID").(uint)

	var merchant models.Merchant
	if err := c.BodyParser(&merchant); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request format",
		})
	}

	merchant.UserID = userID
	merchant.Status = "pending"
	merchant.APIKey = services.GenerateAPIKey() // Generate unique API key
	merchant.VerificationStatus = "pending"

	// Create the merchant profile
	if err := repositories.CreateMerchant(&merchant); err != nil {
		log.Printf("Failed to create merchant: %v", err)
		if strings.Contains(err.Error(), "duplicate key") {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Merchant profile already exists for this user",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create merchant profile",
		})
	}

	// Create default merchant limits
	limits := models.MerchantLimits{
		MerchantID:              merchant.ID,
		DailyTransactionLimit:   10000.0,  // Default daily limit
		MonthlyTransactionLimit: 100000.0, // Default monthly limit
		MinTransactionAmount:    1.0,      // Minimum transaction amount
		MaxTransactionAmount:    1000.0,   // Maximum transaction amount
	}

	if err := repositories.DB.Create(&limits).Error; err != nil {
		log.Printf("Failed to create merchant limits: %v", err)
		// Don't fail the request, just log the error
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":  "Merchant profile created successfully",
		"merchant": merchant,
	})
}

func (h *MerchantHandler) ProcessTransaction(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)
	merchantUserID := claims.UserID

	var input struct {
		Amount      float64 `json:"amount" validate:"required"`
		CustomerID  uint    `json:"customer_id" validate:"required"`
		Description string  `json:"description"`
		PaymentType string  `json:"payment_type"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request format",
		})
	}

	transaction := &models.Transaction{
		TransactionID: fmt.Sprintf("TX-%d-%d", time.Now().Unix(), input.CustomerID),
		Type:          models.TransactionTypeMerchantDirect,
		SenderID:      input.CustomerID,
		ReceiverID:    merchantUserID,
		Amount:        input.Amount,
		Description:   input.Description,
		PaymentType:   input.PaymentType,
		MerchantID:    &merchantUserID,
		Status:        "pending",
		Currency:      "USD",
	}

	processedTx, err := h.merchantService.ProcessTransaction(transaction)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Get merchant details
	merchant, err := repositories.GetMerchantByUserID(merchantUserID)
	if err != nil {
		log.Printf("Failed to get merchant details: %v", err)
		// Return response without merchant details
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message":     "Transaction processed successfully",
			"transaction": processedTx,
		})
	}

	// Only include merchant details if we successfully got them
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":     "Transaction processed successfully",
		"transaction": processedTx,
		"merchant": fiber.Map{
			"id":            merchantUserID,
			"business_name": merchant.BusinessName,
			"business_type": merchant.BusinessType,
		},
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

func (h *MerchantHandler) GenerateQRCode(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)

	var input struct {
		Amount *float64 `json:"amount"`
		Type   string   `json:"type"` // "static" or "dynamic"
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request format",
		})
	}

	qrCode, err := services.NewQRService().GenerateQRCode(claims.UserID, "merchant", input.Type, input.Amount)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"qr_code": qrCode,
	})
}
