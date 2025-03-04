package repositories

import (
	"context"
	"errors"
	"fmt"
	"log"
	"orus/internal/models"
	"time"

	"gorm.io/gorm"
)

// Add these methods to your existing TransactionRepository interface
type TransactionRepository interface {
	// ... existing methods ...
	CreateTransaction(tx *models.Transaction) error
	// Dashboard-specific methods
	GetTransactionStats(userID uint) (count int, volume float64, err error)
	GetLastTransaction(userID uint) (*models.Transaction, error)
	GetRecentMerchants(userID uint, limit int) ([]string, error)
	GetSpendingByCategory(userID uint, since time.Time) (map[string]float64, error)
	GetIncomeByCategory(userID uint, since time.Time) (map[string]float64, error)
	GetUniqueCustomerCount(merchantID uint) (int, error)
	GetTransactionRates(merchantID uint) (successRate, chargebackRate float64, err error)
	GetVolumeOverTime(userID uint, startDate, endDate time.Time) (map[string]float64, error)
	GetTransactionCountByType(userID uint, startDate, endDate time.Time) (map[string]int, error)
	GetMerchantTransactions(merchantID uint, limit, offset int) ([]models.Transaction, int64, error)
	FindByID(id uint) (*models.Transaction, error)
	Update(transaction *models.Transaction) error
	GetDailyTransactionTotal(ctx context.Context, userID uint, start, end time.Time, txType string, total *float64) error
}

func CreateTransaction(tx *models.Transaction) error {
	return DB.Create(tx).Error
}

func GetTransactionByQRCode(qrCodeID string) (*models.Transaction, error) {
	var transaction models.Transaction
	err := DB.Where("qr_code_id = ?", qrCodeID).First(&transaction).Error
	return &transaction, err
}

func UpdateTransaction(tx *models.Transaction) error {
	return DB.Save(tx).Error
}

// Get all transactions for admin with pagination
func GetTransactions(limit, offset int) ([]models.Transaction, int64, error) {
	var transactions []models.Transaction
	var total int64

	DB.Model(&models.Transaction{}).Count(&total)
	result := DB.Limit(limit).Offset(offset).Find(&transactions)

	return transactions, total, result.Error
}

// Get transactions for a specific user with pagination
func GetUserTransactions(userID uint, limit int, offset int) ([]models.Transaction, error) {
	var transactions []models.Transaction
	result := DB.Where("sender_id = ? OR receiver_id = ?", userID, userID).
		Limit(limit).Offset(offset).
		Order("transaction_id DESC"). // Changed from "created_at DESC"
		Find(&transactions)
	return transactions, result.Error
}

func ProcessTransaction(senderID uint, receiverID uint, amount float64, qrCodeID string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		// Validate amount
		if amount <= 0 {
			return errors.New("amount must be greater than zero")
		}

		// Prevent self-transfer
		if senderID == receiverID {
			return errors.New("cannot send money to yourself")
		}

		// Get sender's wallet
		var senderWallet models.Wallet
		err := tx.Where("user_id = ?", senderID).First(&senderWallet).Error
		if err != nil {
			return fmt.Errorf("failed to fetch sender's wallet: %v", err)
		}

		// Check sufficient balance
		if senderWallet.Balance < amount {
			return fmt.Errorf("insufficient funds: available %.2f, requested %.2f", senderWallet.Balance, amount)
		}

		// Get receiver's wallet
		var receiverWallet models.Wallet
		err = tx.Where("user_id = ?", receiverID).First(&receiverWallet).Error
		if err != nil {
			return fmt.Errorf("failed to fetch receiver's wallet: %v", err)
		}

		// Create transaction record
		transaction := &models.Transaction{
			SenderID:   senderID,
			ReceiverID: receiverID,
			Amount:     amount,
			Status:     "pending",
			QRCodeID:   &qrCodeID,
			Type:       "TRANSFER",
		}

		if err := tx.Create(transaction).Error; err != nil {
			return fmt.Errorf("failed to create transaction: %v", err)
		}

		// Update balances
		senderWallet.Balance -= amount
		if err := tx.Model(senderWallet).Update("balance", senderWallet.Balance).Error; err != nil {
			return fmt.Errorf("failed to update sender's wallet: %v", err)
		}

		receiverWallet.Balance += amount
		if err := tx.Model(receiverWallet).Update("balance", receiverWallet.Balance).Error; err != nil {
			return fmt.Errorf("failed to update receiver's wallet: %v", err)
		}

		// Mark transaction as completed
		if err := tx.Model(transaction).Update("status", "completed").Error; err != nil {
			log.Printf("Warning: Failed to update transaction status: %v", err)
		}

		// Invalidate wallet caches
		InvalidateWalletCache(senderID)
		InvalidateWalletCache(receiverID)

		return nil
	})
}

func InvalidateWalletCache(userID uint) {
	key := CacheService.GenerateKey("wallet", "user", userID)
	CacheService.Delete(context.Background(), key)
}

// transactionRepository struct

// FindByID retrieves a transaction by its ID
func (r *transactionRepository) FindByID(id uint) (*models.Transaction, error) {
	var transaction models.Transaction
	err := r.db.First(&transaction, id).Error
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

func (r *transactionRepository) Create(transaction *models.Transaction) error {
	return r.db.Create(transaction).Error
}

func (r *transactionRepository) Update(transaction *models.Transaction) error {
	return r.db.Save(transaction).Error
}

func (r *transactionRepository) GetDailyTransactionTotal(ctx context.Context, userID uint, start, end time.Time, txType string, total *float64) error {
	key := CacheService.GenerateKey("transaction", "daily", map[string]interface{}{
		"user_id": userID,
		"start":   start.Format(time.RFC3339),
		"end":     end.Format(time.RFC3339),
		"type":    txType,
	})

	found, _ := CacheService.Get(ctx, key, total)
	if found {
		return nil
	}

	// If cache miss, proceed to database...
	CacheService.SetWithTTL(ctx, key, *total, 5*time.Minute)
	return nil
}
