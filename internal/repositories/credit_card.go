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
