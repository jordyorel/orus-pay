package repositories

import (
	"orus/internal/models"
)

func CreateCreditCard(card *models.CreateCreditCard) error {
	return DB.Create(card).Error
}

func GetCreditCardsPaginated(limit, offset int) ([]models.CreateCreditCard, int64, error) {
	var creditCards []models.CreateCreditCard
	var total int64

	// Fetch credit cards with pagination
	result := DB.Model(&models.CreateCreditCard{}).Count(&total).Limit(limit).Offset(offset).Find(&creditCards)
	if result.Error != nil {
		return nil, 0, result.Error
	}

	return creditCards, total, nil
}

func GetCreditCardByID(cardID uint) (*models.CreateCreditCard, error) {
	var card models.CreateCreditCard
	err := DB.First(&card, cardID).Error
	return &card, err
}

func GetCreditCardsByUserID(userID uint) ([]models.CreateCreditCard, error) {
	var cards []models.CreateCreditCard
	err := DB.Where("user_id = ?", userID).Find(&cards).Error
	return cards, err
}

func DeleteCreditCard(cardID uint) error {
	return DB.Delete(&models.CreateCreditCard{}, cardID).Error
}
