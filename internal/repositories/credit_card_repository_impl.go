package repositories

import (
	"fmt"
	"orus/internal/models"

	"gorm.io/gorm"
)

type creditCardRepository struct {
	db *gorm.DB
}

func NewCreditCardRepository(db *gorm.DB) CreditCardRepository {
	return &creditCardRepository{
		db: db,
	}
}

func (r *creditCardRepository) GetByID(cardID uint) (*models.CreditCard, error) {
	var card models.CreditCard
	if err := r.db.First(&card, cardID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrCardNotFound
		}
		return nil, fmt.Errorf("failed to get card: %w", err)
	}
	return &card, nil
}

func (r *creditCardRepository) Create(card *models.CreditCard) error {
	return r.db.Create(card).Error
}

func (r *creditCardRepository) Update(card *models.CreditCard) error {
	return r.db.Save(card).Error
}

func (r *creditCardRepository) Delete(cardID uint) error {
	result := r.db.Delete(&models.CreditCard{}, cardID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCardNotFound
	}
	return nil
}

func (r *creditCardRepository) GetByUserID(userID uint) ([]*models.CreditCard, error) {
	var cards []*models.CreditCard
	if err := r.db.Where("user_id = ?", userID).Find(&cards).Error; err != nil {
		return nil, fmt.Errorf("failed to get user cards: %w", err)
	}
	return cards, nil
}

func (r *creditCardRepository) GetDefaultCard(userID uint) (*models.CreditCard, error) {
	var card models.CreditCard
	if err := r.db.Where("user_id = ? AND is_default = ?", userID, true).First(&card).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrCardNotFound
		}
		return nil, fmt.Errorf("failed to get default card: %w", err)
	}
	return &card, nil
}

func (r *creditCardRepository) GetActiveCards(userID uint) ([]*models.CreditCard, error) {
	var cards []*models.CreditCard
	if err := r.db.Where("user_id = ? AND status = ?", userID, "active").Find(&cards).Error; err != nil {
		return nil, fmt.Errorf("failed to get active cards: %w", err)
	}
	return cards, nil
}

func (r *creditCardRepository) UpdateStatus(cardID uint, status string) error {
	result := r.db.Model(&models.CreditCard{}).Where("id = ?", cardID).Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCardNotFound
	}
	return nil
}

func (r *creditCardRepository) SetDefault(cardID uint, isDefault bool) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Get the card to check user_id
		var card models.CreditCard
		if err := tx.First(&card, cardID).Error; err != nil {
			return err
		}

		// Remove default flag from all user's cards
		if err := tx.Model(&models.CreditCard{}).
			Where("user_id = ?", card.UserID).
			Update("is_default", false).Error; err != nil {
			return err
		}

		// Set the new default card
		if isDefault {
			if err := tx.Model(&models.CreditCard{}).
				Where("id = ?", cardID).
				Update("is_default", true).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *creditCardRepository) GetByIDAndUserID(cardID uint, userID uint) (*models.CreditCard, error) {
	var card models.CreditCard
	err := r.db.Where("id = ? AND user_id = ?", cardID, userID).First(&card).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrCardNotFound
		}
		return nil, fmt.Errorf("failed to get card: %w", err)
	}
	return &card, nil
}
