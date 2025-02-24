package handlers

import (
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/utils/response"
	"regexp"

	"orus/internal/utils"

	"orus/internal/services/qr"
	"orus/internal/services/wallet"

	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
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

func (h *UserHandler) GeneratePaymentCode(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)
	qrCode, err := h.qrService.GeneratePaymentCode(c.Context(), claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(fiber.Map{"qr_code": qrCode})
}

func (h *UserHandler) GetReceiveQR(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)

	qr, err := h.qrService.GetUserReceiveQR(c.Context(), claims.UserID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return response.Success(c, "Your QR code for receiving payments", qr)
}

func (h *UserHandler) GetPaymentCodeQR(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)

	qr, err := h.qrService.GetUserPaymentCodeQR(c.Context(), claims.UserID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return response.Success(c, "Your QR code for merchant payments", qr)
}
