package repositories

import (
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
func GetTransactions(limit int, offset int) ([]models.Transaction, error) {
	var transactions []models.Transaction
	result := DB.Limit(limit).Offset(offset).Order("created_at DESC").Find(&transactions)
	return transactions, result.Error
}

// Get transactions for a specific user with pagination
func GetUserTransactions(userID uint, limit int, offset int) ([]models.Transaction, error) {
	var transactions []models.Transaction
	result := DB.Where("sender_id = ? OR receiver_id = ?", userID, userID).
		Limit(limit).Offset(offset).Order("created_at DESC").Find(&transactions)
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
		senderWallet, err := GetWalletByUserID(senderID)
		if err != nil {
			return fmt.Errorf("failed to fetch sender's wallet: %v", err)
		}

		// Check sufficient balance
		if senderWallet.Balance < amount {
			return fmt.Errorf("insufficient funds: available %.2f, requested %.2f", senderWallet.Balance, amount)
		}

		// Get receiver's wallet
		receiverWallet, err := GetWalletByUserID(receiverID)
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
	cacheKey := getWalletCacheKeyByUserID(userID)
	RedisClient.Del(RedisCtx, cacheKey)
}
