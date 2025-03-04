package handlers

import (
	"errors"
	"fmt"
	"log"
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/services/merchant"
	qr "orus/internal/services/qr_code"
	"orus/internal/utils"

	"orus/internal/utils/pagination"
	"orus/internal/utils/response"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type MerchantHandler struct {
	merchantService *merchant.Service
	qrService       qr.Service
	transactionRepo repositories.TransactionRepository
}

func NewMerchantHandler(merchantSvc *merchant.Service, qrSvc qr.Service, transactionRepo repositories.TransactionRepository) *MerchantHandler {
	return &MerchantHandler{
		merchantService: merchantSvc,
		qrService:       qrSvc,
		transactionRepo: transactionRepo,
	}
}

func (h *MerchantHandler) CreateMerchant(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)

	var input struct {
		UserID             uint   `json:"user_id"`
		BusinessName       string `json:"business_name"`
		BusinessType       string `json:"business_type"`
		BusinessAddress    string `json:"business_address"`
		BusinessID         string `json:"business_id"`
		TaxID              string `json:"tax_id"`
		Website            string `json:"website"`
		MerchantCategory   string `json:"merchant_category"`
		LegalEntityType    string `json:"legal_entity_type"`
		RegistrationNumber string `json:"registration_number"`
		YearEstablished    int    `json:"year_established"`
		SupportEmail       string `json:"support_email"`
		SupportPhone       string `json:"support_phone"`
	}

	if err := c.BodyParser(&input); err != nil {
		return utils.BadRequest(c, "Invalid request format")
	}

	// Use the authenticated user's ID if not specified
	if input.UserID == 0 {
		input.UserID = claims.UserID
	}

	// Create merchant profile
	merchant := &models.Merchant{
		UserID:          input.UserID,
		BusinessName:    input.BusinessName,
		BusinessType:    input.BusinessType,
		BusinessAddress: input.BusinessAddress,
		RiskScore:       50, // Default risk score
		ComplianceLevel: "medium_risk",
		Status:          "active",

		// Default limits
		DailyTransactionLimit:   10000,
		MonthlyTransactionLimit: 100000,
		MinTransactionAmount:    1,
		MaxTransactionAmount:    5000,
	}

	// Store additional fields in metadata
	metadata := models.NewJSON(map[string]interface{}{
		"business_id":         input.BusinessID,
		"tax_id":              input.TaxID,
		"website":             input.Website,
		"merchant_category":   input.MerchantCategory,
		"legal_entity_type":   input.LegalEntityType,
		"registration_number": input.RegistrationNumber,
		"year_established":    input.YearEstablished,
		"support_email":       input.SupportEmail,
		"support_phone":       input.SupportPhone,
	})

	// Add metadata field to Merchant model if it doesn't exist
	merchant.Metadata = metadata

	result, err := h.merchantService.CreateMerchant(merchant)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error())
	}

	return response.Success(c, "Merchant profile created successfully", fiber.Map{
		"merchant": result,
	})
}

func (h *MerchantHandler) GetMerchantProfile(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)
	merchant, err := repositories.GetMerchantByUserID(claims.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			defaultMerchant := &models.Merchant{
				UserID:                  claims.UserID,
				BusinessName:            "My Business",
				BusinessType:            "retail",
				Status:                  "pending",
				ComplianceLevel:         "pending",
				RiskScore:               50,
				DailyTransactionLimit:   10000,
				MonthlyTransactionLimit: 100000,
				MinTransactionAmount:    1,
				MaxTransactionAmount:    5000,
			}

			result, err := h.merchantService.CreateMerchant(defaultMerchant)
			if err != nil {
				return response.Error(c, fiber.StatusInternalServerError, "Failed to create merchant profile")
			}

			return response.Success(c, "Default merchant profile created", result)
		}

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

	log.Printf("Attempting to retrieve merchant for userID: %d", claims.UserID)

	var input struct {
		BusinessInfo struct {
			Name               string `json:"name"`
			Type               string `json:"type"`
			RegistrationNumber string `json:"registration_number"`
			TaxID              string `json:"tax_id"`
		} `json:"business_info"`
		ContactInfo struct {
			BusinessEmail string `json:"business_email"`
			BusinessPhone string `json:"business_phone"`
			Website       string `json:"website"`
		} `json:"contact_info"`
		Address struct {
			Street     string `json:"street"`
			Unit       string `json:"unit"`
			City       string `json:"city"`
			PostalCode string `json:"postal_code"`
			Country    string `json:"country"`
		} `json:"address"`
		SettlementInfo struct {
			BankName      string `json:"bank_name"`
			AccountNumber string `json:"account_number"`
			AccountHolder string `json:"account_holder"`
			Currency      string `json:"currency"`
		} `json:"settlement_info"`
		BusinessHours map[string]string `json:"business_hours"`
	}

	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Get existing merchant using the repository directly
	merchant, err := repositories.GetMerchantByUserID(claims.UserID)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Merchant profile not found")
	}

	// Update fields
	merchant.BusinessName = input.BusinessInfo.Name
	merchant.BusinessType = input.BusinessInfo.Type

	// Create full address
	fullAddress := fmt.Sprintf("%s, %s, %s %s, %s",
		input.Address.Street,
		input.Address.Unit,
		input.Address.City,
		input.Address.PostalCode,
		input.Address.Country)
	merchant.BusinessAddress = fullAddress

	// Store all other fields in metadata
	merchant.Metadata = models.NewJSON(map[string]interface{}{
		"business_info":   input.BusinessInfo,
		"contact_info":    input.ContactInfo,
		"address":         input.Address,
		"settlement_info": input.SettlementInfo,
		"business_hours":  input.BusinessHours,
	})

	// Save updated merchant
	if err := repositories.DB.Save(merchant).Error; err != nil {
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

	// Parse the request body
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Call the service to set the webhook URL
	if err := h.merchantService.SetWebhookURL(claims.UserID, input.WebhookURL); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to set webhook URL")
	}

	return response.Success(c, "Webhook URL updated successfully", nil)
}

func (h *MerchantHandler) GetMerchantTransactions(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)
	p := pagination.ParseFromRequest(c)

	transactions, total, err := h.transactionRepo.GetMerchantTransactions(claims.UserID, p.Limit, p.Offset)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to get transactions")
	}

	p.Total = total
	return c.JSON(pagination.Response(p, transactions))
}
