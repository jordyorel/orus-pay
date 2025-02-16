package handlers

import (
	"fmt"
	"log"
	"orus/internal/models"
	"orus/internal/repositories"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Regular expressions for input validation
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
var phoneRegex = regexp.MustCompile(`^\+?[0-9]{7,15}$`) // Allows optional + and 7-15 digits

func RegisterUser(c *fiber.Ctx) error {
	var user models.User

	// Parse request body
	if err := c.BodyParser(&user); err != nil {
		log.Println("Error parsing request body:", err)
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request format"})
	}

	// Validate required fields
	if user.Email == "" || user.Password == "" || user.Phone == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Email, password, and phone are required"})
	}

	// Validate email format
	if !emailRegex.MatchString(user.Email) {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid email format"})
	}

	// Validate password strength
	if len(user.Password) < 8 || !hasSpecialChar(user.Password) {
		return c.Status(400).JSON(fiber.Map{"error": "Password must be at least 8 characters long and include a special character"})
	}

	// Validate phone number format
	if !phoneRegex.MatchString(user.Phone) {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid phone number format"})
	}

	// Sanitize user inputs
	user.Email = strings.TrimSpace(user.Email)
	user.Phone = strings.TrimSpace(user.Phone)

	// Check for existing user (including soft-deleted users)
	var existingUser models.User
	err := repositories.DB.Unscoped().Where("email = ? OR phone = ?", user.Email, user.Phone).First(&existingUser).Error
	if err == nil { // User exists (active or soft-deleted)
		if existingUser.DeletedAt.Valid { // Soft-deleted user found
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "That email/phone is already associated with an account. Please log in."})
		} else { // Active user found
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "User with this email/phone already exists"})
		}
	} else if err != gorm.ErrRecordNotFound { // Database error (not "record not found")
		log.Println("Error checking for existing user:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Error hashing password:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Internal server error"})
	}
	user.Password = string(hashedPassword)

	user.Role = "user"

	// Save user to the database
	if err := repositories.CreateUser(&user); err != nil {
		if strings.Contains(err.Error(), "uni_users_email") {
			return c.Status(400).JSON(fiber.Map{"error": "User with this email already exists"})
		}
		log.Println("Error creating user:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Registration failed"})
	}

	// 🔹 Automatically Create Wallet for the User with a Fixed QR Code
	wallet := models.Wallet{
		UserID:   user.ID,
		Balance:  0.00,
		Currency: "USD",
		QRCodeID: fmt.Sprintf("orus://pay?user_id=%d", user.ID),
	}

	if err := repositories.CreateWallet(&wallet); err != nil {
		log.Println("Error creating wallet:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create wallet"})
	}

	// Log success (without sensitive data)
	// log.Printf("User registered successfully: Email: %s, Wallet created, QR Code ID: %s", user.Email, wallet.QRCodeID)

	// Return success response
	return c.JSON(fiber.Map{
		"message": "User registered successfully",
		"user": fiber.Map{
			"id":    user.ID,
			"email": user.Email,
			"phone": user.Phone,
			"role":  user.Role,
		},
		"wallet": fiber.Map{
			"id":         wallet.ID,
			"balance":    wallet.Balance,
			"currency":   wallet.Currency,
			"qr_code_id": wallet.QRCodeID,
		},
	})
}

// Helper function to check if a password contains at least one special character
func hasSpecialChar(s string) bool {
	specialCharRegex := regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`)
	return specialCharRegex.MatchString(s)
}
