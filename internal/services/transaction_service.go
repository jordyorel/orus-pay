package services

import (
	"errors"
	"fmt"
	"math"
	"orus/internal/models"
	"orus/internal/repositories"
	"time"

	"gorm.io/gorm"
)

var (
	ErrHighRiskTransaction        = errors.New("transaction risk too high")
	ErrUnsupportedTransactionType = errors.New("unsupported transaction type")
)

type TransactionService struct {
	walletService *WalletService
	riskService   *RiskService
}

func NewTransactionService() *TransactionService {
	return &TransactionService{
		walletService: NewWalletService(),
		riskService:   NewRiskService(),
	}
}

func (s *TransactionService) ProcessTransaction(tx *models.Transaction) (*models.Transaction, error) {
	// Round amount to 2 decimal places
	tx.Amount = math.Round(tx.Amount*100) / 100

	// Risk assessment
	riskScore := s.riskService.AssessTransaction(tx)
	if riskScore > highRiskThreshold {
		return nil, ErrHighRiskTransaction
	}

	// Validate transaction type
	switch tx.Type {
	case models.TransactionTypeTopup,
		models.TransactionTypeWithdrawal,
		models.TransactionTypeQRPayment,
		models.TransactionTypeMerchantDirect: // Add this case
		// Valid transaction type
	default:
		return nil, fmt.Errorf("unsupported transaction type: %s", tx.Type)
	}

	// Process based on transaction type
	switch tx.Type {
	case models.TransactionTypeMerchantDirect:
		// Handle merchant direct payment
		err := s.walletService.Debit(tx.SenderID, tx.Amount)
		if err != nil {
			return nil, err
		}
		err = s.walletService.Credit(tx.ReceiverID, tx.Amount)
		if err != nil {
			// Rollback sender debit if receiver credit fails
			_ = s.walletService.Credit(tx.SenderID, tx.Amount)
			return nil, err
		}
		tx.Status = "completed"

	case models.TransactionTypeTopup:
		// ... existing topup logic ...
	case models.TransactionTypeWithdrawal:
		// ... existing withdrawal logic ...
	case models.TransactionTypeQRPayment:
		err := s.walletService.Debit(tx.SenderID, tx.Amount)
		if err != nil {
			return nil, err
		}
		err = s.walletService.Credit(tx.ReceiverID, tx.Amount)
		if err != nil {
			_ = s.walletService.Credit(tx.SenderID, tx.Amount)
			return nil, err
		}
		tx.Status = "completed"
	}

	// Save the transaction
	if err := repositories.CreateTransaction(tx); err != nil {
		return nil, err
	}

	return tx, nil
}

func (s *TransactionService) ProcessP2PTransfer(senderID, receiverID uint, amount float64, description string) (*models.Transaction, error) {
	tx := &models.Transaction{
		TransactionID: fmt.Sprintf("TX-%d-%d", time.Now().Unix(), senderID),
		Type:          "P2P_TRANSFER",
		SenderID:      senderID,
		ReceiverID:    receiverID,
		Amount:        amount,
		Description:   description,
		Status:        "pending",
	}

	// Process in DB transaction
	err := repositories.DB.Transaction(func(db *gorm.DB) error {
		// Debit sender
		if err := NewWalletService().Debit(senderID, amount); err != nil {
			return err
		}
		// Credit receiver
		if err := NewWalletService().Credit(receiverID, amount); err != nil {
			return err
		}
		// Record transaction
		tx.Status = "completed"
		return db.Create(tx).Error
	})

	if err != nil {
		return nil, err
	}
	return tx, nil
}
