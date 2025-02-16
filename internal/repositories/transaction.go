package repositories

import "orus/internal/models"

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
