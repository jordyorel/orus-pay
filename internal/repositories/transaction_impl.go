package repositories

import (
	"fmt"
	"orus/internal/models"
	"time"

	"gorm.io/gorm"
)

type transactionRepository struct {
	db *gorm.DB
}

func (r *transactionRepository) GetTransactionStats(userID uint) (count int, volume float64, err error) {
	result := r.db.Model(&models.Transaction{}).
		Where("sender_id = ? OR receiver_id = ?", userID, userID).
		Select("COUNT(*) as count, COALESCE(SUM(amount), 0) as volume").
		Row()

	err = result.Scan(&count, &volume)
	return
}

func (r *transactionRepository) GetLastTransaction(userID uint) (*models.Transaction, error) {
	var tx models.Transaction
	err := r.db.Where("sender_id = ? OR receiver_id = ?", userID, userID).
		Order("created_at DESC").
		First(&tx).Error
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (r *transactionRepository) GetRecentMerchants(userID uint, limit int) ([]string, error) {
	var merchants []string
	err := r.db.Model(&models.Transaction{}).
		Where("sender_id = ?", userID).
		Select("DISTINCT merchant_name").
		Where("merchant_name != ''").
		Limit(limit).
		Pluck("merchant_name", &merchants).Error
	return merchants, err
}

func (r *transactionRepository) GetSpendingByCategory(userID uint, since time.Time) (map[string]float64, error) {
	type Result struct {
		Category string
		Total    float64
		Type     string
	}
	var results []Result

	err := r.db.Model(&models.Transaction{}).
		Where("sender_id = ? AND created_at >= ? AND type NOT IN (?, ?)",
			userID, since, "top_up", models.TransactionTypeRefund).
		Select("category, SUM(amount) as total, type").
		Group("category, type").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	spending := make(map[string]float64)
	for _, r := range results {
		category := r.Category
		if category == "" {
			category = getCategoryFromType(r.Type)
		}
		spending[category] = r.Total
	}
	return spending, nil
}

func (r *transactionRepository) GetIncomeByCategory(userID uint, since time.Time) (map[string]float64, error) {
	type Result struct {
		Type  string
		Total float64
	}
	var results []Result

	err := r.db.Model(&models.Transaction{}).
		Where("(receiver_id = ? OR sender_id = ?) AND created_at >= ? AND type IN (?, ?, ?)",
			userID, userID, since,
			"top_up",
			models.TransactionTypeRefund,
			models.TransactionTypeP2PTransfer).
		Select("type, SUM(amount) as total").
		Group("type").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	income := make(map[string]float64)
	for _, r := range results {
		switch r.Type {
		case "top_up":
			income["Top Up"] = r.Total
		case models.TransactionTypeRefund:
			income["Refund"] = r.Total
		case models.TransactionTypeP2PTransfer:
			income["Received Transfer"] = r.Total
		default:
			income["Other Income"] = r.Total
		}
	}
	return income, nil
}

func (r *transactionRepository) GetUniqueCustomerCount(merchantID uint) (int, error) {
	var count int64
	err := r.db.Model(&models.Transaction{}).
		Where("receiver_id = ?", merchantID).
		Distinct("sender_id").
		Count(&count).Error
	return int(count), err
}

func (r *transactionRepository) GetTransactionRates(merchantID uint) (successRate, chargebackRate float64, err error) {
	// Get total transaction count
	var total, successful, chargebacks int64

	err = r.db.Model(&models.Transaction{}).
		Where("receiver_id = ?", merchantID).
		Count(&total).Error
	if err != nil {
		return 0, 0, err
	}

	// Get successful transactions
	err = r.db.Model(&models.Transaction{}).
		Where("receiver_id = ? AND status = ?", merchantID, "completed").
		Count(&successful).Error
	if err != nil {
		return 0, 0, err
	}

	// Get chargeback count
	err = r.db.Model(&models.Transaction{}).
		Where("receiver_id = ? AND status = ?", merchantID, "chargeback").
		Count(&chargebacks).Error
	if err != nil {
		return 0, 0, err
	}

	if total > 0 {
		successRate = float64(successful) / float64(total) * 100
		chargebackRate = float64(chargebacks) / float64(total) * 100
	}

	return successRate, chargebackRate, nil
}

func (r *transactionRepository) GetVolumeOverTime(userID uint, startDate, endDate time.Time) (map[string]float64, error) {
	type Result struct {
		Date  string
		Total float64
	}
	var results []Result

	err := r.db.Model(&models.Transaction{}).
		Where("(sender_id = ? OR receiver_id = ?) AND created_at BETWEEN ? AND ?",
			userID, userID, startDate, endDate).
		Select("DATE(created_at) as date, SUM(amount) as total").
		Group("DATE(created_at)").
		Order("date").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	volume := make(map[string]float64)
	for _, r := range results {
		volume[r.Date] = r.Total
	}
	return volume, nil
}

func (r *transactionRepository) GetTransactionCountByType(userID uint, startDate, endDate time.Time) (map[string]int, error) {
	type Result struct {
		Type  string
		Count int
	}
	var results []Result

	err := r.db.Model(&models.Transaction{}).
		Where("(sender_id = ? OR receiver_id = ?) AND created_at BETWEEN ? AND ?",
			userID, userID, startDate, endDate).
		Select("type, COUNT(*) as count").
		Group("type").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[string]int)
	for _, r := range results {
		counts[r.Type] = r.Count
	}
	return counts, nil
}

func (r *transactionRepository) GetMerchantTransactions(merchantID uint, limit, offset int) ([]models.Transaction, int64, error) {
	var transactions []models.Transaction
	var total int64

	// Get total count
	if err := r.db.Model(&models.Transaction{}).
		Where("receiver_id = ? AND type IN (?, ?, ?)",
			merchantID,
			"merchant_scan",
			"merchant_direct",
			models.TransactionTypeMerchantScan).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated transactions with merchant details
	err := r.db.Where("receiver_id = ? AND type IN (?, ?, ?)",
		merchantID,
		"merchant_scan",
		"merchant_direct",
		models.TransactionTypeMerchantScan).
		Joins("LEFT JOIN merchants ON merchants.user_id = transactions.receiver_id").
		Select("transactions.*, merchants.business_name as merchant_name, merchants.business_type as merchant_category").
		Order("transactions.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&transactions).Error

	// Set additional fields
	for i := range transactions {
		transactions[i].MerchantID = &merchantID
		if transactions[i].Category == "" {
			transactions[i].Category = "Sale"
		}
		// Set transaction reference fields
		if transactions[i].TransactionID == "" {
			transactions[i].TransactionID = fmt.Sprintf("MTXN-%d-%d", merchantID, transactions[i].ID)
		}
		if transactions[i].Reference == "" {
			transactions[i].Reference = fmt.Sprintf("MREF-%d-%d", merchantID, transactions[i].ID)
		}
		if transactions[i].PaymentType == "" {
			switch transactions[i].Type {
			case "merchant_scan":
				transactions[i].PaymentType = "qr_scan"
				transactions[i].PaymentMethod = "wallet"
			case "merchant_direct":
				transactions[i].PaymentType = "direct"
				transactions[i].PaymentMethod = "wallet"
			default:
				transactions[i].PaymentType = "standard"
				transactions[i].PaymentMethod = "wallet"
			}
		}
	}

	return transactions, total, err
}

func NewTransactionRepository(db *gorm.DB) TransactionRepository {
	return &transactionRepository{
		db: db,
	}
}

func UpdateTransactionCategories() error {
	// Set default categories based on transaction types
	return DB.Exec(`
		UPDATE transactions 
		SET category = CASE
			WHEN type = 'TOPUP' THEN 'Deposit'
			WHEN type = 'WITHDRAWAL' THEN 'Withdrawal'
			WHEN type = 'P2P_TRANSFER' THEN 'Transfer'
			WHEN type = 'QR_PAYMENT' THEN 'Payment'
			WHEN type = 'merchant_direct' THEN 'Shopping'
			WHEN type = 'merchant_scan' THEN 'Shopping'
			WHEN type = 'refund' THEN 'Refund'
			ELSE 'Other'
		END
		WHERE category IS NULL OR category = ''
	`).Error
}

// Helper function to map transaction types to categories
func getCategoryFromType(txType string) string {
	switch txType {
	case "debit":
		return "Withdrawal"
	case models.TransactionTypeQRPayment:
		return "QR Payment"
	case models.TransactionTypeMerchantDirect:
		return "Shopping"
	case models.TransactionTypeMerchantScan:
		return "Shopping"
	case models.TransactionTypeP2PTransfer:
		return "Transfer"
	case models.TransactionTypeWithdrawal:
		return "Withdrawal"
	default:
		return "Other"
	}
}
