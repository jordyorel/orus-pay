package handlers

import (
	"log"
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"orus/internal/models"
	"orus/internal/repositories"
)

const (
	maxTransactionLimit = 100 // Maximum allowed transactions per page
)

// In handlers/transaction.go
func ProcessTransaction(c *fiber.Ctx) error {
	var input struct {
		ReceiverID uint    `json:"receiver_id" validate:"required"`
		Amount     float64 `json:"amount" validate:"required,gt=0"`
		QRCodeID   string  `json:"qr_code_id"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if input.ReceiverID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Receiver ID is required",
		})
	}

	if input.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Amount must be greater than zero",
		})
	}

	// Get sender from claims
	claims := c.Locals("claims").(*models.UserClaims)
	senderID := claims.UserID

	// Process transaction
	err := repositories.ProcessTransaction(senderID, input.ReceiverID, input.Amount, input.QRCodeID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Transaction processed successfully",
	})
}

func GetUserTransactions(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	// Enforce pagination limits
	if limit > maxTransactionLimit {
		limit = maxTransactionLimit
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	transactions, err := repositories.GetUserTransactions(userID, limit, offset)
	if err != nil {
		log.Printf("Transaction history error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to retrieve transactions"})
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
