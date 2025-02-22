package services

import (
	"fmt"
	"orus/internal/repositories"

	"errors"

	"gorm.io/gorm"
)

type WalletService struct{}

func NewWalletService() *WalletService {
	return &WalletService{}
}

func (s *WalletService) Credit(userID uint, amount float64) error {
	return repositories.DB.Transaction(func(tx *gorm.DB) error {
		wallet, err := repositories.GetWalletByUserID(userID)
		if err != nil {
			return fmt.Errorf("failed to get wallet: %w", err)
		}
		wallet.Balance += amount
		return tx.Save(wallet).Error
	})
}

func (s *WalletService) Debit(userID uint, amount float64) error {
	return repositories.DB.Transaction(func(tx *gorm.DB) error {
		wallet, err := repositories.GetWalletByUserIDForUpdate(userID)
		if err != nil {
			return fmt.Errorf("failed to get wallet: %w", err)
		}

		if wallet.Balance < amount {
			return errors.New("insufficient funds")
		}

		wallet.Balance -= amount
		return tx.Save(wallet).Error
	})
}
