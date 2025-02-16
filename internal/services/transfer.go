package services

import (
	"fmt"
	"orus/internal/models"
	"orus/internal/repositories"

	"gorm.io/gorm"
)

func TransferFunds(senderID, receiverID uint, amount float64) error {
	return repositories.DB.Transaction(func(tx *gorm.DB) error {
		var sender, receiver models.Wallet

		if err := tx.Model(&sender).Where("user_id = ?", senderID).
			Update("balance", gorm.Expr("balance - ?", amount)).Error; err != nil {
			return fmt.Errorf("insufficient funds")
		}

		if err := tx.Model(&receiver).Where("user_id = ?", receiverID).
			Update("balance", gorm.Expr("balance + ?", amount)).Error; err != nil {
			return err
		}

		return nil
	})
}
