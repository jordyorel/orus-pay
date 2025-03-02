package dispute

import (
	"errors"
	"orus/internal/models"
	"orus/internal/repositories"

	"gorm.io/gorm"
)

type Service struct {
	repo            repositories.DisputeRepository
	transactionRepo repositories.TransactionRepository
	userRepo        repositories.UserRepository
	db              *gorm.DB
}

func NewService(repo repositories.DisputeRepository, transactionRepo repositories.TransactionRepository, userRepo repositories.UserRepository, db *gorm.DB) *Service {
	return &Service{repo: repo, transactionRepo: transactionRepo, userRepo: userRepo, db: db}
}

func (s *Service) FileDispute(transactionID, userID uint, reason string) (*models.Dispute, error) {
	// Retrieve the transaction to check user involvement
	transaction, err := s.transactionRepo.FindByID(transactionID)
	if err != nil {
		return nil, errors.New("transaction not found")
	}

	// Check if the user is either the sender or receiver
	if transaction.SenderID != userID && transaction.ReceiverID != userID {
		return nil, errors.New("user is not involved in this transaction")
	}

	// Check if MerchantID is valid
	if transaction.MerchantID == nil {
		return nil, errors.New("transaction is not associated with a merchant")
	}

	// Check if a dispute already exists for this transaction
	exists, err := s.repo.ExistsByTransactionID(transactionID)
	if err != nil {
		return nil, err
	}
	if exists {
		// Check if the existing dispute is refunded
		refunded, err := s.repo.IsRefunded(transactionID)
		if err != nil {
			return nil, err
		}
		if refunded {
			return nil, errors.New("a dispute has already been filed and refunded for this transaction")
		}
		return nil, errors.New("a dispute has already been filed for this transaction")
	}

	// Create the dispute
	dispute := &models.Dispute{
		TransactionID: transactionID,
		MerchantID:    *transaction.MerchantID,
		UserID:        userID,
		Reason:        reason,
	}

	if err := s.repo.Create(dispute); err != nil {
		return nil, err
	}
	return dispute, nil
}

func (s *Service) GetDisputes(merchantID uint) ([]models.Dispute, error) {
	return s.repo.FindByMerchantID(merchantID)
}

func (s *Service) GetMerchantDisputes(merchantID uint) ([]models.Dispute, error) {
	return s.repo.FindByMerchantID(merchantID)
}

func (s *Service) ProcessRefund(disputeID uint) error {
	// Check if the dispute exists
	dispute, err := s.repo.FindByID(disputeID)
	if err != nil {
		return errors.New("dispute not found")
	}

	// Check if the dispute is already refunded
	if dispute.Refunded {
		return errors.New("dispute has already been refunded")
	}

	// Retrieve the transaction associated with the dispute
	transaction, err := s.transactionRepo.FindByID(dispute.TransactionID)
	if err != nil {
		return errors.New("transaction not found")
	}

	// Determine the roles
	var senderID, receiverID uint
	if transaction.SenderID == dispute.UserID {
		senderID = transaction.SenderID     // Customer
		receiverID = transaction.ReceiverID // Merchant
	} else {
		senderID = transaction.ReceiverID // Customer
		receiverID = transaction.SenderID // Merchant
	}

	// Start a transaction
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Credit to the customer
		if err := s.updateUserBalance(receiverID, transaction.Amount); err != nil {
			return err
		}

		// Deduct from the merchant
		if err := s.updateUserBalance(senderID, -transaction.Amount); err != nil {
			return err
		}

		// Update the dispute to mark it as refunded
		dispute.Refunded = true
		if err := s.repo.Update(dispute); err != nil {
			return err
		}

		// Record the refund transaction (optional)
		refundTransaction := &models.Transaction{
			SenderID:   senderID,
			ReceiverID: receiverID,
			Amount:     transaction.Amount,
			Status:     "completed", // or "refunded"
			Type:       "REFUND",    // Indicate this is a refund transaction
		}
		if err := s.transactionRepo.CreateTransaction(refundTransaction); err != nil {
			return err
		}

		return nil
	})

	return err
}

func (s *Service) ProcessChargeback(disputeID uint) error {
	// Check if the dispute exists
	dispute, err := s.repo.FindByID(disputeID)
	if err != nil {
		return errors.New("dispute not found")
	}

	// Check if the dispute is already processed
	if dispute.Status != "pending" {
		return errors.New("dispute cannot be charged back")
	}

	// Retrieve the transaction associated with the dispute
	transaction, err := s.transactionRepo.FindByID(dispute.TransactionID)
	if err != nil {
		return errors.New("transaction not found")
	}

	// Start a transaction
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Update the transaction status to chargeback
		transaction.Status = "chargeback"
		if err := s.transactionRepo.Update(transaction); err != nil {
			return err
		}

		// Adjust the balances
		if err := s.updateUserBalance(transaction.ReceiverID, -transaction.Amount); err != nil {
			return err
		}
		if err := s.updateUserBalance(transaction.SenderID, transaction.Amount); err != nil {
			return err
		}

		// Update the dispute status
		dispute.Status = "charged_back"
		if err := s.repo.Update(dispute); err != nil {
			return err
		}

		return nil
	})

	return err
}

// Example method to update user balance
func (s *Service) updateUserBalance(userID uint, amount float64) error {
	// Retrieve the current balance
	currentBalance, err := s.userRepo.GetBalance(userID)
	if err != nil {
		return err
	}

	// Calculate the new balance
	newBalance := currentBalance + amount

	// Update the user's balance in the database
	if err := s.userRepo.UpdateBalance(userID, newBalance); err != nil {
		return err
	}

	return nil
}
