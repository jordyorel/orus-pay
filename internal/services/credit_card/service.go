package credit_card

import (
	"context"
	"errors"
	"fmt"
	"log"
	"orus/internal/models"
	"orus/internal/repositories"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/stripe/stripe-go/v72"
)

var (
	ErrCardNotFound        = errors.New("card not found")
	ErrCardNotActive       = errors.New("card not active")
	ErrCardNotBelongToUser = errors.New("card does not belong to user")
)

type service struct {
	repo repositories.CreditCardRepository
}

func NewService(repo repositories.CreditCardRepository) Service {
	return &service{
		repo: repo,
	}
}

// Core validation methods
func (s *service) ValidateCard(ctx context.Context, userID, cardID uint) error {
	card, err := s.GetCard(ctx, cardID)
	if err != nil {
		return err
	}

	if card.UserID != userID {
		return ErrCardNotBelongToUser
	}

	if card.Status != "active" {
		return ErrCardNotActive
	}

	return nil
}

func (s *service) GetCard(ctx context.Context, cardID uint) (*models.CreditCard, error) {
	card, err := s.repo.GetByID(cardID)
	if err != nil {
		if err == repositories.ErrCardNotFound {
			return nil, ErrCardNotFound
		}
		return nil, fmt.Errorf("failed to get card: %w", err)
	}
	return card, nil
}

// Tokenization methods
func (s *service) TokenizeCreditCard(card models.CreateCardInput) (*models.VisaCardToken, error) {
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
		// ... rest of test cards
	}

	// Check if this is a test token
	if strings.HasPrefix(card.CardNumber, "tok_") {
		cardType := "Unknown"
		switch card.CardNumber {
		case "tok_visa", "tok_visa_debit":
			cardType = "Visa"
		case "tok_mastercard", "tok_mastercard_2":
			cardType = "Mastercard"
			// ... rest of token types
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

	// Validate card number and expiry
	if !s.isValidCardNumber(card.CardNumber) {
		return nil, errors.New("invalid card number: failed Luhn check")
	}

	expiryMonth, err := strconv.Atoi(card.ExpiryMonth)
	if err != nil {
		return nil, errors.New("invalid expiry month format")
	}
	expiryYear, err := strconv.Atoi(card.ExpiryYear)
	if err != nil {
		return nil, errors.New("invalid expiry year format")
	}

	if !s.isValidExpiryDate(expiryMonth, expiryYear) {
		return nil, errors.New("card is expired or has invalid expiry date")
	}

	return nil, errors.New("direct card tokenization is not supported - please use Stripe Elements or Mobile SDK")
}

// Helper methods
func (s *service) isValidCardNumber(cardNumber string) bool {
	var sum int
	shouldDouble := false

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

	return sum%10 == 0
}

func (s *service) isValidExpiryDate(month, year int) bool {
	if month < 1 || month > 12 {
		return false
	}

	currentYear, currentMonth, _ := time.Now().Date()
	if year < currentYear || (year == currentYear && month < int(currentMonth)) {
		return false
	}

	return true
}

func (s *service) LinkCard(userID uint, input models.CreateCardInput) (*models.CreditCard, error) {
	token, err := s.TokenizeCreditCard(input)
	if err != nil {
		return nil, err
	}

	card := &models.CreditCard{
		UserID:      userID,
		CardNumber:  token.Token, // Store token instead of actual number
		CardType:    token.CardType,
		ExpiryMonth: input.ExpiryMonth,
		ExpiryYear:  input.ExpiryYear,
		LastFour:    input.CardNumber[len(input.CardNumber)-4:],
		Status:      "active",
	}

	if err := s.repo.Create(card); err != nil {
		return nil, err
	}

	return card, nil
}
