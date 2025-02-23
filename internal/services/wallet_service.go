package services

import (
	"orus/internal/models"
	"orus/internal/repositories"

	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type WalletService struct{}

func NewWalletService() *WalletService {
	return &WalletService{}
}

func (s *WalletService) Credit(userID uint, amount float64) error {
	wallet, err := repositories.GetWalletByUserIDForUpdate(userID)
	if err != nil {
		return err
	}

	wallet.Balance += amount
	return repositories.DB.Save(wallet).Error
}

func (s *WalletService) Debit(userID uint, amount float64) error {
	wallet, err := repositories.GetWalletByUserIDForUpdate(userID)
	if err != nil {
		return err
	}

	if wallet.Balance < amount {
		return errors.New("insufficient balance")
	}

	wallet.Balance -= amount
	return repositories.DB.Save(wallet).Error
}

func (s *WalletService) GetWallet(userID uint) (*models.Wallet, error) {
	// Always get fresh data from DB for wallet balance
	wallet, err := repositories.GetWalletByUserIDForUpdate(userID)
	if err != nil {
		return nil, err
	}

	// Ensure balance is non-negative
	if wallet.Balance < 0 {
		wallet.Balance = 0
		if err := repositories.UpdateWallet(wallet); err != nil {
			return nil, err
		}
	}

	return wallet, nil
}

func (s *WalletService) TopUp(userID uint, amount float64, cardID uint) error {
	// Get wallet with lock
	wallet, err := repositories.GetWalletByUserIDForUpdate(userID)
	if err != nil {
		return err
	}

	// Verify card
	card, err := repositories.GetCreditCardByID(cardID)
	if err != nil {
		return err
	}
	if card.UserID != userID {
		return errors.New("card does not belong to user")
	}

	// Create transaction record
	tx := &models.Transaction{
		TransactionID: fmt.Sprintf("TX-%d-%d", time.Now().Unix(), userID),
		Type:          "TOPUP",
		ReceiverID:    userID,
		SenderID:      0, // System transaction
		Amount:        amount,
		PaymentType:   "CARD",
		CardID:        &cardID,
		Status:        "pending",
	}

	// Process in DB transaction
	err = repositories.DB.Transaction(func(db *gorm.DB) error {
		// Update wallet balance
		wallet.Balance += amount
		if err := repositories.UpdateWallet(wallet); err != nil {
			return err
		}

		// Update transaction status
		tx.Status = "completed"
		if err := db.Create(tx).Error; err != nil {
			return err
		}

		return nil
	})

	return err
}
