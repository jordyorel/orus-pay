package handlers

import (
	"orus/internal/models"
	"orus/internal/repositories"
	"regexp"

	"orus/internal/utils"

	qr "orus/internal/services/qr_code"
	"orus/internal/services/wallet"

	"fmt"
	"log"
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

const (
	maxTransactionLimit = 100 // Maximum allowed transactions per page
)

// Regular expressions for input validation
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
var phoneRegex = regexp.MustCompile(`^\+?[0-9]{7,15}$`) // Allows optional + and 7-15 digits

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
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Validate role
	validRoles := map[string]bool{
		"user":       true,
		"merchant":   true,
		"enterprise": true,
	}
	if !validRoles[input.Role] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid role. Must be one of: user, merchant, enterprise",
		})
	}

	// Validate password
	if len(input.Password) < 8 || !utils.HasSpecialChar(input.Password) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Password must be at least 8 characters and contain special characters",
		})
	}

	// Validate email
	if !emailRegex.MatchString(input.Email) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid email format",
		})
	}

	// Validate phone
	if !phoneRegex.MatchString(input.Phone) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid phone format",
		})
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

	// Get wallet
	wallet, err := h.walletService.GetWallet(c.Context(), createdUser.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get wallet info",
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
	userID := c.Locals("userID").(uint)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit > maxTransactionLimit {
		limit = maxTransactionLimit
	}

	transactions, err := repositories.GetUserTransactions(userID, limit, (page-1)*limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch transactions",
		})
	}

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

	return c.JSON(fiber.Map{
		"transactions": sanitized,
		"page":         page,
		"limit":        limit,
		"total":        len(sanitized),
	})
}
