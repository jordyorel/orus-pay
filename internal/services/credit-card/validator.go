package creditcard

import (
	"errors"
	"strconv"
	"time"
)

func (s *Service) validateCardInput(card CreateCardInput) error {
	if card.CardNumber == "" {
		return errors.New("card number is required")
	}
	if card.ExpiryMonth == "" || card.ExpiryYear == "" {
		return errors.New("expiry date is required")
	}

	month, err := strconv.Atoi(card.ExpiryMonth)
	if err != nil || month < 1 || month > 12 {
		return errors.New("invalid expiry month")
	}

	year, err := strconv.Atoi(card.ExpiryYear)
	if err != nil {
		return errors.New("invalid expiry year")
	}

	now := time.Now()
	if year < now.Year() || (year == now.Year() && month < int(now.Month())) {
		return errors.New("card has expired")
	}

	return nil
}
