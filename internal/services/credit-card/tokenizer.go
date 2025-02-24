package creditcard

import (
	"errors"
	"strings"
)

// Tokenizer handles credit card tokenization
type Tokenizer interface {
	TokenizeCard(card CreateCardInput) (*TokenizedCard, error)
}

type DefaultTokenizer struct {
	testCards map[string]struct {
		token    string
		cardType string
	}
}

func NewTokenizer() Tokenizer {
	return &DefaultTokenizer{
		testCards: map[string]struct {
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
		},
	}
}

func (t *DefaultTokenizer) TokenizeCard(card CreateCardInput) (*TokenizedCard, error) {
	// Check if this is a test token
	if strings.HasPrefix(card.CardNumber, "tok_") {
		cardType := t.getCardTypeFromToken(card.CardNumber)
		return &TokenizedCard{
			Token:    card.CardNumber,
			CardType: cardType,
			LastFour: "4242", // Default for test tokens
			IssuedBy: "Test Issuer",
		}, nil
	}

	// Check if this is a test card number
	if testCard, isTestCard := t.testCards[card.CardNumber]; isTestCard {
		return &TokenizedCard{
			Token:    testCard.token,
			CardType: testCard.cardType,
			LastFour: card.CardNumber[len(card.CardNumber)-4:],
			IssuedBy: "Test Bank",
		}, nil
	}

	// Validate card number using Luhn algorithm
	if !isValidCardNumber(card.CardNumber) {
		return nil, errors.New("invalid card number: failed Luhn check")
	}

	// For production cards, return error indicating direct tokenization is not supported
	return nil, errors.New("direct card tokenization is not supported - please use Stripe Elements or Mobile SDK")
}

func (t *DefaultTokenizer) getCardTypeFromToken(token string) string {
	switch token {
	case "tok_visa", "tok_visa_debit":
		return "Visa"
	case "tok_mastercard", "tok_mastercard_2":
		return "Mastercard"
	case "tok_amex":
		return "American Express"
	case "tok_discover":
		return "Discover"
	case "tok_diners":
		return "Diners Club"
	default:
		return "Unknown"
	}
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
