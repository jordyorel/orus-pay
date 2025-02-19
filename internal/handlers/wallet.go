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

	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/stripe/stripe-go/v72"
)

// TopUpWallet adds funds to the user's wallet.
func TopUpWallet(c *fiber.Ctx) error {
	// Extract user ID from JWT
	userID, ok := c.Locals("userID").(uint)
	if !ok || userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	// Parse request body
	var request struct {
		Amount   float64 `json:"amount"`
		CardID   uint    `json:"card_id"`  // ID of the saved card to use
		Currency string  `json:"currency"` // Optional: defaults to USD
	}

	if err := c.BodyParser(&request); err != nil {
		log.Println("Error parsing request body:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request format"})
	}

	// Validate amount
	if request.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid top-up amount"})
	}

	// Set default currency if not provided
	if request.Currency == "" {
		request.Currency = "usd"
	}

	// Fetch user's wallet
	wallet, err := repositories.GetWalletByUserID(userID)
	if err != nil {
		log.Println("Wallet not found for user:", userID)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Wallet not found"})
	}

	// Fetch the card details
	card, err := repositories.GetCreditCardByID(request.CardID)
	if err != nil {
		log.Println("Card not found:", err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Card not found"})
	}

	// Verify card belongs to user
	if card.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Card does not belong to user"})
	}

	// Create payment intent with Stripe
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	// amountInCents := int64(request.Amount * 100) // Convert to cents

	// For test cards, simulate payment processing
	var paymentStatus string
	cardNumberStr := fmt.Sprintf("%v", card.CardNumber) // Convert card number to string for comparison
	switch cardNumberStr {
	case "tok_visa", "4242424242424242": // Always succeeds
		paymentStatus = "succeeded"
	case "tok_visa_declined", "4000000000000002": // Always fails
		paymentStatus = "failed"
	case "tok_visa_insufficient_funds", "4000000000009995": // Insufficient funds
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Insufficient funds on card"})
	default:
		paymentStatus = "succeeded" // Default to success for other test cards
	}

	// Create transaction record
	transaction := &models.Transaction{
		ReceiverID:  userID,
		SenderID:    0, // System transaction
		Amount:      request.Amount,
		Status:      "pending",
		Type:        "TOPUP",
		PaymentType: "CARD",
		CardID:      &request.CardID,
		Currency:    request.Currency,
	}

	if err := repositories.CreateTransaction(transaction); err != nil {
		log.Println("Error creating transaction:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create transaction"})
	}

	// Process based on payment status
	if paymentStatus == "succeeded" {
		// Update wallet balance
		wallet.Balance += request.Amount
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
			// Don't return error since the top-up was successful
		}

		return c.JSON(fiber.Map{
			"message":        "Top-up successful",
			"transaction_id": transaction.ID,
			"wallet": fiber.Map{
				"id":       wallet.ID,
				"balance":  wallet.Balance,
				"currency": wallet.Currency,
			},
		})
	} else {
		// Payment failed
		transaction.Status = "failed"
		repositories.UpdateTransaction(transaction)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":          "Payment failed",
			"transaction_id": transaction.ID,
		})
	}
}

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

	// Check if wallet has sufficient funds
	if wallet.Balance < request.Amount {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Insufficient funds"})
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
		ReceiverID:  0, // System transaction
		Amount:      request.Amount,
		Status:      "pending",
		Type:        "WITHDRAWAL",
		PaymentType: "CARD",
		CardID:      &request.CardID,
		Currency:    request.Currency,
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
		wallet.Balance -= request.Amount
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
			wallet.Balance += request.Amount
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
