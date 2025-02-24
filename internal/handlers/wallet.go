package handlers

import (
	"fmt"
	"log"
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/services"

	"github.com/gofiber/fiber/v2"
)

// TopUpWallet adds funds to the user's wallet without fees
func TopUpWallet(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)
	if claims == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized access",
		})
	}

	var input struct {
		Amount float64 `json:"amount" validate:"required,gt=0"`
		CardID uint    `json:"card_id" validate:"required"`
	}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request format",
		})
	}

	walletService := services.NewWalletService()
	err := walletService.TopUp(claims.UserID, input.Amount, input.CardID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Top up successful",
		"amount":  input.Amount,
	})
}

// WithdrawToCard handles withdrawal with fees
func WithdrawToCard(c *fiber.Ctx) error {
	// Extract user ID from JWT
	userID, ok := c.Locals("userID").(uint)
	if !ok || userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// Parse request body
	var request struct {
		Amount   float64 `json:"amount"`
		CardID   uint    `json:"card_id"`
		Currency string  `json:"currency"`
	}

	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request format"})
	}

	// Validate amount
	if request.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid withdrawal amount"})
	}

	// Set default currency if not provided
	if request.Currency == "" {
		request.Currency = "usd"
	}

	// Fetch user's wallet
	wallet, err := repositories.GetWalletByUserID(userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Wallet not found"})
	}

	// Calculate withdrawal fee (e.g., 1.5%)
	fee := request.Amount * 0.015
	totalDeduction := request.Amount + fee

	if wallet.Balance < totalDeduction {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Insufficient balance including fees",
			"fee":   fee,
		})
	}

	// Fetch the card details
	card, err := repositories.GetCreditCardByID(request.CardID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Card not found"})
	}

	// Verify card belongs to user
	if card.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Card does not belong to user"})
	}

	// Create transaction record
	transaction := &models.Transaction{
		SenderID:    userID,
		ReceiverID:  0, // System
		Amount:      request.Amount,
		Fee:         fee,
		Status:      "pending",
		Type:        "WITHDRAWAL",
		PaymentType: "CARD",
		CardID:      &request.CardID,
	}

	if err := repositories.CreateTransaction(transaction); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create transaction"})
	}

	// For test cards, simulate payout processing
	var payoutStatus string
	cardNumberStr := fmt.Sprintf("%v", card.CardNumber) // Convert card number to string for comparison
	switch cardNumberStr {
	case "tok_visa", "4242424242424242": // Always succeeds
		payoutStatus = "succeeded"
	case "tok_visa_declined", "4000000000000002": // Always fails
		payoutStatus = "failed"
	default:
		payoutStatus = "succeeded" // Default to success for other test cards
	}

	if payoutStatus == "succeeded" {
		// Update wallet balance
		wallet.Balance -= totalDeduction
		if err := repositories.UpdateWallet(wallet); err != nil {
			log.Println("Error updating wallet:", err)
			transaction.Status = "failed"
			repositories.UpdateTransaction(transaction)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update wallet"})
		}

		// Update transaction status
		transaction.Status = "completed"
		if err := repositories.UpdateTransaction(transaction); err != nil {
			log.Println("Error updating transaction:", err)
			// Try to rollback the wallet balance
			wallet.Balance += totalDeduction
			if rollbackErr := repositories.UpdateWallet(wallet); rollbackErr != nil {
				log.Printf("Critical error: Failed to rollback wallet balance after transaction status update failed. Original error: %v, Rollback error: %v", err, rollbackErr)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":          "Critical error occurred during withdrawal. Please contact support with transaction ID",
					"transaction_id": transaction.ID,
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":          "Withdrawal failed - transaction status update error",
				"transaction_id": transaction.ID,
			})
		}

		return c.JSON(fiber.Map{
			"message":        "Withdrawal successful",
			"transaction_id": transaction.ID,
			"wallet": fiber.Map{
				"id":       wallet.ID,
				"balance":  wallet.Balance,
				"currency": wallet.Currency,
			},
		})
	} else {
		// Payout failed
		transaction.Status = "failed"
		repositories.UpdateTransaction(transaction)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":          "Withdrawal failed",
			"transaction_id": transaction.ID,
		})
	}
}

// GetWallet retrieves the user's wallet details.
func GetWallet(c *fiber.Ctx) error {
	claims := c.Locals("claims").(*models.UserClaims)
	if claims == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized access",
		})
	}

	walletService := services.NewWalletService()
	wallet, err := walletService.GetWallet(claims.UserID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Wallet not found",
		})
	}

	return c.JSON(fiber.Map{
		"wallet": wallet,
	})
}
