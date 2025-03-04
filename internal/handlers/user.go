package handlers

import (
	"orus/internal/models"
	qr "orus/internal/services/qr_code"
	"orus/internal/services/user"
	"orus/internal/services/wallet"
	"orus/internal/utils/pagination"
	"orus/internal/utils/response"

	"github.com/gofiber/fiber/v2"
)

// const maxTransactionLimit = 100 // Maximum allowed transactions per page

// UserHandler manages user-related HTTP requests including registration,
// profile management, and user settings.
type UserHandler struct {
	userService   user.Service
	walletService wallet.Service
	qrService     qr.Service
}

func NewUserHandler(userService user.Service, walletService wallet.Service, qrService qr.Service) *UserHandler {
	return &UserHandler{
		userService:   userService,
		walletService: walletService,
		qrService:     qrService,
	}
}

// RegisterUser handles new user registration requests.
// It creates a new user account, associated wallet, and generates initial QR codes.
//
// Request body should contain:
// - name: User's full name
// - email: User's email address
// - phone: User's phone number (optional)
// - password: User's password
//
// Returns:
// - 200: Successful registration with user details and initial QR codes
// - 400: Invalid request format or validation errors
// - 500: Internal server error during user creation
func (h *UserHandler) RegisterUser(c *fiber.Ctx) error {
	var input models.CreateUserInput
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	user, err := h.userService.Create(&input)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error())
	}

	// Create wallet for the new user
	wallet, err := h.walletService.CreateWallet(c.Context(), user.ID, "USD")
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to create wallet")
	}

	// Generate QR codes
	receiveQR, err := h.qrService.GetUserReceiveQR(c.Context(), user.ID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to generate receive QR")
	}

	paymentQR, err := h.qrService.GetUserPaymentCodeQR(c.Context(), user.ID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to generate payment QR")
	}

	// Hide sensitive data
	user.Password = ""
	user.WalletID = &wallet.ID
	user.Wallet = wallet

	return response.Success(c, "User registered successfully", fiber.Map{
		"user":            user,
		"receive_qr":      receiveQR,
		"payment_code_qr": paymentQR,
	})
}

// GetProfile returns the user's profile
func (h *UserHandler) GetProfile(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)

	user, err := h.userService.GetByID(claims.UserID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to get user profile")
	}

	// Hide sensitive data
	user.Password = ""

	return response.Success(c, "Profile retrieved", user)
}

// UpdateProfile updates the user's profile
func (h *UserHandler) UpdateProfile(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)

	var input models.UpdateUserInput
	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	user, err := h.userService.GetByID(claims.UserID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to get user")
	}

	// Update fields
	if input.Name != "" {
		user.Name = input.Name
	}
	if input.Phone != "" {
		user.Phone = input.Phone
	}

	if err := h.userService.Update(user); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to update profile")
	}

	return response.Success(c, "Profile updated", nil)
}

// ChangePassword changes the user's password
func (h *UserHandler) ChangePassword(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)

	var input struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	if err := c.BodyParser(&input); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.userService.ChangePassword(claims.UserID, input.OldPassword, input.NewPassword); err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error())
	}

	return response.Success(c, "Password changed successfully", nil)
}

// GetUserTransactions returns the user's transactions
func (h *UserHandler) GetUserTransactions(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)

	p := pagination.ParseFromRequest(c)

	transactions, total, err := h.userService.GetTransactions(claims.UserID, p.Limit, p.Offset)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to fetch transactions")
	}

	p.Total = total
	return c.JSON(pagination.Response(p, transactions))
}
