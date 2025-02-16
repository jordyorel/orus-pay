package handlers

import (
	"errors"
	"fmt"
	"log"
	"orus/internal/models"
	"orus/internal/repositories"
	"orus/internal/services"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

// LinkCreditCard links a credit card to the user's account.
func LinkCreditCard(c *fiber.Ctx) error {
	// Extract user ID from JWT
	userID := c.Locals("userID")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// Parse JSON body
	var card models.CreateCreditCard
	if err := c.BodyParser(&card); err != nil {
		log.Println("Error parsing request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request format"})
	}

	// Validate input
	if err := validateCardInput(card); err != nil {
		log.Println("Validation error:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	// Call tokenization service
	tokenizedCard, err := services.TokenizeCreditCard(card)
	if err != nil {
		log.Println("Tokenization failed:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Card tokenization failed"})
	}

	// Save tokenized card
	cardRecord := models.CreateCreditCard{
		UserID:      userID.(uint),
		CardNumber:  tokenizedCard.Token, // Store the token, not the card number
		CardType:    tokenizedCard.CardType,
		ExpiryMonth: card.ExpiryMonth,
		ExpiryYear:  card.ExpiryYear,
	}

	if err := repositories.CreateCreditCard(&cardRecord); err != nil {
		log.Printf("Failed to save card record: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to link card"})
	}

	return c.JSON(fiber.Map{
		"message":   "Credit card linked successfully",
		"token":     tokenizedCard.Token,
		"card_type": tokenizedCard.CardType,
		"expiry":    fmt.Sprintf("%s/%s", card.ExpiryMonth, card.ExpiryYear),
	})
}

// validateCardInput validates the credit card input.
func validateCardInput(card models.CreateCreditCard) error {
	if card.CardNumber == "" {
		return errors.New("card number is required")
	}
	if card.ExpiryMonth == "" || card.ExpiryYear == "" {
		return errors.New("expiry date is required")
	}

	// Convert expiry dates to integers for validation
	month, err := strconv.Atoi(card.ExpiryMonth)
	if err != nil || month < 1 || month > 12 {
		return errors.New("invalid expiry month")
	}

	year, err := strconv.Atoi(card.ExpiryYear)
	if err != nil {
		return errors.New("invalid expiry year")
	}

	// Validate expiry date
	now := time.Now()
	if year < now.Year() || (year == now.Year() && month < int(now.Month())) {
		return errors.New("card has expired")
	}

	return nil
}

// TopUpWallet adds funds to the user's wallet.
func TopUpWallet(c *fiber.Ctx) error {
	// Extract user ID from JWT (set by AuthMiddleware)
	userID, ok := c.Locals("userID").(uint)
	if !ok || userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// Parse request body
	var request struct {
		Amount float64 `json:"amount"`
	}

	if err := c.BodyParser(&request); err != nil {
		log.Println("Error parsing request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request format"})
	}

	// Validate amount
	if request.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid top-up amount"})
	}

	// Fetch user's wallet
	wallet, err := repositories.GetWalletByUserID(userID)
	if err != nil {
		log.Println("Wallet not found for user:", userID)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Wallet not found"})
	}

	// Create top-up transaction record
	transaction := &models.Transaction{
		ReceiverID: userID, // User receiving the money
		SenderID:   0,      // 0 indicates system/top-up transaction
		Amount:     request.Amount,
		Status:     "pending",
		Type:       "TOPUP",
	}

	if err := repositories.CreateTransaction(transaction); err != nil {
		log.Println("Error creating top-up transaction:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create transaction record"})
	}

	// Update balance
	wallet.Balance += request.Amount
	if err := repositories.UpdateWallet(wallet); err != nil {
		log.Println("Error updating wallet balance:", err)
		// Mark transaction as failed since wallet update failed
		transaction.Status = "failed"
		repositories.UpdateTransaction(transaction)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update wallet balance"})
	}

	// Mark transaction as completed
	transaction.Status = "completed"
	if err := repositories.UpdateTransaction(transaction); err != nil {
		log.Println("Error updating transaction status:", err)
		// Note: We don't return an error here since the top-up was successful
	}

	log.Printf("User %d topped up %.2f %s", userID, request.Amount, wallet.Currency)

	// Return success response
	return c.JSON(fiber.Map{
		"message":        "Wallet topped up successfully",
		"transaction_id": transaction.ID,
		"wallet": fiber.Map{
			"id":       wallet.ID,
			"balance":  wallet.Balance,
			"currency": wallet.Currency,
		},
	})
}

// GetWallet retrieves the user's wallet details.
func GetWallet(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uint)
	if !ok || userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// Fetch the user's wallet
	wallet, err := repositories.GetWalletByUserID(userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Wallet not found"})
	}

	return c.JSON(fiber.Map{"wallet": wallet})
}
