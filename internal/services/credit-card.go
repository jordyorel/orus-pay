package services

import (
	"errors"
	"fmt"
	"log"
	"orus/internal/models"
	"os"
	"strings"
	"time"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/token"
)

func TokenizeCreditCard(card models.CreateCreditCard) (*models.VisaCardToken, error) {
	log.Printf("Processing card: %s", card.CardNumber)

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	// Handle test tokens differently
	if strings.HasPrefix(card.CardNumber, "tok_") {
		// For test tokens, we can return the token directly
		cardType := "Unknown"
		switch card.CardNumber {
		case "tok_visa":
			cardType = "Visa"
		case "tok_mastercard":
			cardType = "Mastercard"
		case "tok_amex":
			cardType = "American Express"
		case "tok_discover":
			cardType = "Discover"
		}

		return &models.VisaCardToken{
			Token:    card.CardNumber,
			CardType: cardType,
			Expiry:   fmt.Sprintf("%s/%s", card.ExpiryMonth, card.ExpiryYear),
		}, nil
	}

	// For real card numbers, proceed with normal validation and tokenization
	if !isValidCardNumber(card.CardNumber) {
		return nil, errors.New("invalid card number: failed validation check")
	}

	params := &stripe.TokenParams{
		Card: &stripe.CardParams{
			Number:   &card.CardNumber,
			ExpMonth: &card.ExpiryMonth,
			ExpYear:  &card.ExpiryYear,
		},
	}

	stripeToken, err := token.New(params)
	if err != nil {
		log.Printf("Stripe tokenization error: %v", err)
		return nil, fmt.Errorf("stripe tokenization failed: %v", err)
	}

	return &models.VisaCardToken{
		Token:    stripeToken.ID,
		CardType: string(stripeToken.Card.Brand),
		Expiry:   fmt.Sprintf("%s/%s", card.ExpiryMonth, card.ExpiryYear),
	}, nil
}

// Luhn Algorithm: Used to validate credit card numbers
func isValidCardNumber(cardNumber string) bool {
	var sum int
	shouldDouble := false

	// Iterate over the digits of the card number from right to left
	for i := len(cardNumber) - 1; i >= 0; i-- {
		digit := int(cardNumber[i] - '0')

		if shouldDouble {
			digit = digit * 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		shouldDouble = !shouldDouble
	}

	// Card is valid if the sum is a multiple of 10
	return sum%10 == 0
}

// Check if the expiry date is valid (MM/YYYY)
func isValidExpiryDate(month, year int) bool {
	// Ensure the month is between 1 and 12
	if month < 1 || month > 12 {
		return false
	}

	// Get the current date
	currentYear, currentMonth, _ := time.Now().Date()

	// Compare the expiry year and month with the current date
	if year < currentYear || (year == currentYear && month < int(currentMonth)) {
		return false // Card is expired if the expiry date is in the past
	}

	// Otherwise, it's a valid expiry date
	return true
}
