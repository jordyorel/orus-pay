package transfer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"orus/internal/models"
	"orus/internal/repositories"

	"gorm.io/gorm"
)

// service implements the transfer Service interface.
type service struct {
	walletSvc WalletService
	notifier  NotificationService
}

// NewService creates a new transfer service instance.
func NewService(walletSvc WalletService, notifier NotificationService) Service {
	return &service{
		walletSvc: walletSvc,
		notifier:  notifier,
	}
}

// Transfer moves funds between two user wallets.
func (s *service) Transfer(ctx context.Context, senderID, receiverID uint, amount float64, description string) (*models.Transaction, error) {
	if senderID == receiverID {
		return nil, errors.New("cannot transfer to self")
	}
	if amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}

	if err := s.walletSvc.ValidateBalance(ctx, senderID, amount); err != nil {
		return nil, err
	}

	tx := &models.Transaction{
		Type:          models.TransactionTypeP2PTransfer,
		SenderID:      senderID,
		ReceiverID:    receiverID,
		Amount:        amount,
		Description:   description,
		Status:        "pending",
		TransactionID: fmt.Sprintf("P2P-%d-%d-%d", senderID, receiverID, time.Now().UnixNano()),
	}

	err := repositories.DB.Transaction(func(dbTx *gorm.DB) error {
		if err := s.walletSvc.Debit(ctx, senderID, amount); err != nil {
			return err
		}
		if err := s.walletSvc.Credit(ctx, receiverID, amount); err != nil {
			return err
		}
		tx.Status = "completed"
		return dbTx.Create(tx).Error
	})
	if err != nil {
		return nil, err
	}

	if s.notifier != nil {
		_ = s.notifier.SendTransferNotification(ctx, senderID, tx)
		_ = s.notifier.SendTransferNotification(ctx, receiverID, tx)
	}

	return tx, nil
}
