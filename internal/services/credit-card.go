package services

import (
	"errors"
	"fmt"
	"log"
	"orus/internal/models"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/stripe/stripe-go/v72"
)

func TokenizeCreditCard(card models.CreateCreditCard) (*models.VisaCardToken, error) {
	log.Printf("Processing card: %s", card.CardNumber)

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	// Map of test card numbers to their corresponding Stripe test tokens
	testCards := map[string]struct {
		token    string
		cardType string
	}{
		"4242424242424242": {"tok_visa", "Visa"},
		"4000056655665556": {"tok_visa_debit", "Visa Debit"},
		"5555555555554444": {"tok_mastercard", "Mastercard"},
		"2223003122003222": {"tok_mastercard_2", "Mastercard"},
		"378282246310005":  {"tok_amex", "American Express"},
		"6011111111111117": {"tok_discover", "Discover"},
		"3056930009020004": {"tok_diners", "Diners Club"},
		"36227206271667":   {"tok_diners", "Diners Club"},
	}

	// Check if this is a test token
	if strings.HasPrefix(card.CardNumber, "tok_") {
		cardType := "Unknown"
		switch card.CardNumber {
		case "tok_visa", "tok_visa_debit":
			cardType = "Visa"
		case "tok_mastercard", "tok_mastercard_2":
			cardType = "Mastercard"
		case "tok_amex":
			cardType = "American Express"
		case "tok_discover":
			cardType = "Discover"
		case "tok_diners":
			cardType = "Diners Club"
		}

		return &models.VisaCardToken{
			Token:    card.CardNumber,
			CardType: cardType,
			Expiry:   fmt.Sprintf("%s/%s", card.ExpiryMonth, card.ExpiryYear),
		}, nil
	}

	// Check if this is a test card number
	if testCard, isTestCard := testCards[card.CardNumber]; isTestCard {
		return &models.VisaCardToken{
			Token:    testCard.token,
			CardType: testCard.cardType,
			Expiry:   fmt.Sprintf("%s/%s", card.ExpiryMonth, card.ExpiryYear),
		}, nil
	}

	// Validate card number using Luhn algorithm
	if !isValidCardNumber(card.CardNumber) {
		return nil, errors.New("invalid card number: failed Luhn check")
	}

	// Convert expiry month/year to integers for validation
	expiryMonth, err := strconv.Atoi(card.ExpiryMonth)
	if err != nil {
		return nil, errors.New("invalid expiry month format")
	}
	expiryYear, err := strconv.Atoi(card.ExpiryYear)
	if err != nil {
		return nil, errors.New("invalid expiry year format")
	}

	// Validate expiry date
	if !isValidExpiryDate(expiryMonth, expiryYear) {
		return nil, errors.New("card is expired or has invalid expiry date")
	}

	// For production cards, return error indicating direct tokenization is not supported
	return nil, errors.New("direct card tokenization is not supported - please use Stripe Elements or Mobile SDK for security compliance")
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
