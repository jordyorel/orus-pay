package repositories

import (
	"orus/internal/models"
)

func CreateCreditCard(card *models.CreditCard) error {
	result := DB.Table("credit_cards").Create(card)
	return result.Error
}

func GetCreditCardsPaginated(limit, offset int) ([]models.CreditCard, int64, error) {
	var creditCards []models.CreditCard
	var total int64

	// Fetch credit cards with pagination
	result := DB.Model(&models.CreditCard{}).Count(&total).Limit(limit).Offset(offset).Find(&creditCards)
	if result.Error != nil {
		return nil, 0, result.Error
	}

	return creditCards, total, nil
}

func GetCreditCardByID(cardID uint) (*models.CreditCard, error) {
	var card models.CreditCard
	err := DB.First(&card, cardID).Error
	return &card, err
}

func GetCreditCardsByUserID(userID uint) ([]models.CreditCard, error) {
	var cards []models.CreditCard
	err := DB.Where("user_id = ?", userID).Find(&cards).Error
	return cards, err
}

func DeleteCreditCard(cardID uint) error {
	return DB.Delete(&models.CreditCard{}, cardID).Error
}
