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
	var request struct {
		Amount   float64 `json:"amount"`
		QRCodeID string  `json:"qr_code_id"`
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request format"})
	}

	if request.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Amount must be greater than zero"})
	}

	if request.QRCodeID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "QR code is required"})
	}

	senderID := c.Locals("userID").(uint)

	// Fetch receiver's wallet using QRCodeID
	receiverWallet, err := repositories.GetWalletByQRCodeID(request.QRCodeID)
	if err != nil {
		log.Printf("Error in transaction processing: %v", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Invalid or non-existent QR code"})
	}

	if receiverWallet.UserID == senderID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot send money to yourself"})
	}

	// Fetch sender's wallet
	senderWallet, err := repositories.GetWalletByUserID(senderID)
	if err != nil {
		log.Printf("Error fetching sender wallet: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch sender's wallet"})
	}

	if senderWallet.Balance < request.Amount {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":             "Insufficient funds",
			"available_balance": senderWallet.Balance,
			"requested_amount":  request.Amount,
		})
	}

	// Create transaction
	transaction := models.Transaction{
		SenderID:   senderID,
		ReceiverID: receiverWallet.UserID,
		Amount:     request.Amount,
		Status:     "pending",
		QRCodeID:   request.QRCodeID,
		Type:       "TRANSFER",
	}

	if err := repositories.CreateTransaction(&transaction); err != nil {
		log.Printf("Error creating transaction: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create transaction"})
	}

	// Update balances
	senderWallet.Balance -= transaction.Amount
	if err := repositories.UpdateWallet(senderWallet); err != nil {
		transaction.Status = "failed"
		repositories.UpdateTransaction(&transaction)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update sender's wallet"})
	}

	receiverWallet.Balance += transaction.Amount
	if err := repositories.UpdateWallet(receiverWallet); err != nil {
		// Roll back sender's wallet
		senderWallet.Balance += transaction.Amount
		repositories.UpdateWallet(senderWallet) // Should handle this error too

		transaction.Status = "failed"
		repositories.UpdateTransaction(&transaction)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update receiver's wallet"})
	}

	if err := repositories.UpdateWallet(senderWallet); err != nil {
		log.Printf("Error updating sender wallet: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update sender's wallet"})
	}

	if err := repositories.UpdateWallet(receiverWallet); err != nil {
		log.Printf("Error updating receiver wallet: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update receiver's wallet"})
	}

	// Mark transaction as completed
	transaction.Status = "completed"
	if err := repositories.UpdateTransaction(&transaction); err != nil {
		log.Printf("Error updating transaction status: %v", err)
		// Transaction and wallet updates succeeded, so we'll still return success
		// but we should log this for monitoring
	}
	repositories.UpdateTransaction(&transaction)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":        "Transaction completed successfully",
		"transaction_id": transaction.ID,
		"amount":         transaction.Amount,
		"new_balance":    senderWallet.Balance,
		"recipient": map[string]interface{}{
			"qr_code_id": receiverWallet.QRCodeID,
		},
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
