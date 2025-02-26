package handlers

import (
	"fmt"
	"log"
	"math"
	"orus/internal/models"
	"orus/internal/repositories"
	qr "orus/internal/services/qr_code"
	"orus/internal/services/wallet"
	"orus/internal/utils"
	"orus/internal/validation"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

const maxTransactionLimit = 100 // Maximum allowed transactions per page

type UserHandler struct {
	qrService     qr.Service
	walletService wallet.Service
}

func NewUserHandler(qrSvc qr.Service, walletSvc wallet.Service) *UserHandler {
	return &UserHandler{
		qrService:     qrSvc,
		walletService: walletSvc,
	}
}

func (h *UserHandler) RegisterUser(c *fiber.Ctx) error {
	var input models.CreateUserInput
	if err := c.BodyParser(&input); err != nil {
		return utils.BadRequest(c, "Invalid request body")
	}

	v := validation.New()
	v.UserRegistration(&input)
	if !v.Valid() {
		// Get first error from the map
		for _, msg := range v.Errors {
			return utils.BadRequest(c, msg)
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Password hashing failed"})
	}

	// Create base user
	user := &models.User{
		Name:     input.Name,
		Email:    input.Email,
		Phone:    input.Phone,
		Password: string(hashedPassword),
		Role:     input.Role,
		Status:   "active",
		MerchantProfileStatus: func() string {
			if input.Role == "merchant" {
				return "pending_completion"
			}
			return "not_applicable"
		}(),
	}

	createdUser, _, err := repositories.CreateUser(user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Create wallet for the new user
	wallet, err := h.walletService.CreateWallet(c.Context(), createdUser.ID, "USD")
	if err != nil {
		log.Printf("Failed to create wallet for user %d: %v", createdUser.ID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create wallet",
		})
	}

	// Get both QR codes
	receiveQR, err := h.qrService.GetUserReceiveQR(c.Context(), createdUser.ID)
	if err != nil {
		log.Printf("Failed to generate receive QR: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to generate receive QR: %v", err),
		})
	}

	paymentCodeQR, err := h.qrService.GetUserPaymentCodeQR(c.Context(), createdUser.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate payment code QR",
		})
	}

	// Update user with wallet info
	createdUser.WalletID = &wallet.ID
	createdUser.Wallet = wallet // Include the wallet object
	createdUser.Password = ""

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":         "User registered successfully",
		"user":            createdUser,
		"receive_qr":      receiveQR,
		"payment_code_qr": paymentCodeQR,
	})
}

func (h *UserHandler) GetUserTransactions(c *fiber.Ctx) error {
	claims, ok := c.Locals("claims").(*models.UserClaims)
	if !ok {
		return utils.Unauthorized(c, "invalid claims")
	}
	userID := claims.UserID

	pagination := utils.GetPagination(c, 1, maxTransactionLimit)

	transactions, err := repositories.GetUserTransactions(userID, pagination.Limit, pagination.Offset)
	if err != nil {
		log.Printf("Error fetching transactions: %v", err)
		return utils.InternalError(c, "Failed to fetch transactions")
	}

	pagination.SetTotal(int64(len(transactions)))

	// Sanitize transaction data
	sanitized := make([]map[string]interface{}, len(transactions))
	for i, t := range transactions {
		sanitized[i] = map[string]interface{}{
			"id":         t.ID,
			"amount":     math.Round(t.Amount*100) / 100,
			"status":     t.Status,
			"type":       t.Type,
			"created_at": t.CreatedAt,
		}
	}

	return c.JSON(utils.NewPaginatedResponse(sanitized, pagination))
}
