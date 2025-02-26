package creditcard

import (
	"errors"
	"fmt"
	"log"
	"orus/internal/models"
	"orus/internal/repositories"
)

type Service struct {
	tokenizer Tokenizer
}

func NewService() *Service {
	return &Service{
		tokenizer: NewTokenizer(),
	}
}

func (s *Service) LinkCard(userID uint, input CreateCardInput) (*models.CreditCard, error) {
	if err := s.validateCardInput(input); err != nil {
		return nil, err
	}

	tokenizedCard, err := s.tokenizer.TokenizeCard(input)
	if err != nil {
		log.Println("Tokenization failed:", err)
		return nil, fmt.Errorf("card tokenization failed: %w", err)
	}

	cardRecord := &models.CreditCard{
		UserID:      userID,
		CardNumber:  tokenizedCard.Token,
		CardType:    tokenizedCard.CardType,
		ExpiryMonth: input.ExpiryMonth,
		ExpiryYear:  input.ExpiryYear,
		Status:      "active",
	}

	if err := repositories.CreateCreditCard(cardRecord); err != nil {
		return nil, fmt.Errorf("failed to save card: %w", err)
	}

	return cardRecord, nil
}

func (s *Service) GetUserCards(userID uint) ([]models.CreditCard, error) {
	return repositories.GetCreditCardsByUserID(userID)
}

func (s *Service) DeleteCard(userID uint, cardID uint) error {
	card, err := repositories.GetCreditCardByID(cardID)
	if err != nil {
		return err
	}

	if card.UserID != userID {
		return errors.New("card does not belong to user")
	}

	return repositories.DeleteCreditCard(cardID)
}
