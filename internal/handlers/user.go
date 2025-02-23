package handlers

import (
	"orus/internal/models"
	"orus/internal/repositories"
	"regexp"

	"orus/internal/utils"

	"orus/internal/services"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

// Regular expressions for input validation
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
var phoneRegex = regexp.MustCompile(`^\+?[0-9]{7,15}$`) // Allows optional + and 7-15 digits

type UserHandler struct {
	qrService *services.QRService
}

func NewUserHandler() *UserHandler {
	return &UserHandler{
		qrService: services.NewQRService(),
	}
}

func RegisterUser(c *fiber.Ctx) error {
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

	createdUser, qrCode, err := repositories.CreateUser(user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// For merchant role, return instructions
	if input.Role == "merchant" {
		createdUser.Password = ""
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"user":      createdUser,
			"message":   "Please complete your merchant profile by sending a POST request to /api/merchant with your business details",
			"next_step": "/api/merchant",
			"static_qr": qrCode,
		})
	}

	createdUser.Password = ""
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message":   "User registered successfully",
		"user":      createdUser,
		"static_qr": qrCode,
	})
}

func (h *UserHandler) GeneratePaymentCode(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)
	qrCode, err := h.qrService.GeneratePaymentCode(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(fiber.Map{"qr_code": qrCode})
}

func (h *UserHandler) GenerateReceiveCode(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)
	qrCode, err := h.qrService.GenerateQRCode(claims.UserID, "user", "static", nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(fiber.Map{"qr_code": qrCode})
}

func (h *UserHandler) GetReceiveCode(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)
	qrCode, err := repositories.GetUserStaticQR(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get receive code",
		})
	}
	return c.JSON(fiber.Map{"qr_code": qrCode})
}

func (h *UserHandler) GetReceiveQR(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)

	qr, err := services.NewQRService().GeneratePaymentCode(claims.UserID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"qr_code": qr,
		"message": "Use this QR code to receive payments",
	})
}
