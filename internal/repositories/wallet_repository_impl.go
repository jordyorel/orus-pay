package repositories

import (
	"context"
	"fmt"
	"orus/internal/models"
	"time"

	"gorm.io/gorm"
)

type walletRepository struct {
	db *gorm.DB
}

func NewWalletRepository(db *gorm.DB) WalletRepository {
	return &walletRepository{
		db: db,
	}
}

func (r *walletRepository) Create(wallet *models.Wallet) error {
	result := r.db.Create(wallet)
	if result.Error != nil {
		return fmt.Errorf("failed to create wallet: %w", result.Error)
	}
	return nil
}

func (r *walletRepository) GetByID(id uint) (*models.Wallet, error) {
	var wallet models.Wallet
	if err := r.db.First(&wallet, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrWalletNotFound
		}
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}
	return &wallet, nil
}

func (r *walletRepository) GetByUserID(userID uint) (*models.Wallet, error) {
	var wallet models.Wallet
	if err := r.db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrWalletNotFound
		}
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}
	return &wallet, nil
}

func (r *walletRepository) Update(wallet *models.Wallet) error {
	result := r.db.Save(wallet)
	if result.Error != nil {
		return fmt.Errorf("failed to update wallet: %w", result.Error)
	}
	return nil
}

func (r *walletRepository) Delete(id uint) error {
	result := r.db.Delete(&models.Wallet{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete wallet: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrWalletNotFound
	}
	return nil
}

func (r *walletRepository) CreateTransaction(tx *models.Transaction) error {
	result := r.db.Create(tx)
	if result.Error != nil {
		return fmt.Errorf("failed to create transaction: %w", result.Error)
	}
	return nil
}

func (r *walletRepository) GetTransactionByID(id uint) (*models.Transaction, error) {
	var tx models.Transaction
	if err := r.db.First(&tx, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrInvalidTransaction
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	return &tx, nil
}

func (r *walletRepository) GetTransactionHistory(ctx context.Context, walletID uint, limit, offset int, dest interface{}) error {
	err := r.db.WithContext(ctx).
		Where("sender_id = ? OR receiver_id = ?", walletID, walletID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(dest).Error
	if err != nil {
		return fmt.Errorf("failed to get transaction history: %w", err)
	}
	return nil
}

func (r *walletRepository) GetDailyTransactionTotal(ctx context.Context, userID uint, start, end time.Time, txType string, total *float64) error {
	err := r.db.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("sender_id = ? AND type = ? AND created_at BETWEEN ? AND ?", userID, txType, start, end).
		Select("COALESCE(SUM(amount), 0)").
		Scan(total).Error
	if err != nil {
		return fmt.Errorf("failed to get daily transaction total: %w", err)
	}
	return nil
}

func (r *walletRepository) GetMonthlyTransactionTotal(ctx context.Context, userID uint, start, end time.Time, txType string, total *float64) error {
	err := r.db.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("sender_id = ? AND type = ? AND created_at BETWEEN ? AND ?", userID, txType, start, end).
		Select("COALESCE(SUM(amount), 0)").
		Scan(total).Error
	if err != nil {
		return fmt.Errorf("failed to get monthly transaction total: %w", err)
	}
	return nil
}

func (r *walletRepository) ExecuteInTransaction(fn func(WalletRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := &walletRepository{db: tx}
		return fn(txRepo)
	})
}

func (r *walletRepository) BulkCreate(wallets []*models.Wallet) error {
	result := r.db.Create(wallets)
	if result.Error != nil {
		return fmt.Errorf("failed to bulk create wallets: %w", result.Error)
	}
	return nil
}

func (r *walletRepository) BulkUpdate(wallets []*models.Wallet) error {
	for _, wallet := range wallets {
		if err := r.Update(wallet); err != nil {
			return err
		}
	}
	return nil
}

func (r *walletRepository) UpdateStatus(walletID uint, status string) error {
	result := r.db.Model(&models.Wallet{}).Where("id = ?", walletID).Update("status", status)
	if result.Error != nil {
		return fmt.Errorf("failed to update wallet status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrWalletNotFound
	}
	return nil
}

func (r *walletRepository) GetWalletsByStatus(status string) ([]*models.Wallet, error) {
	var wallets []*models.Wallet
	if err := r.db.Where("status = ?", status).Find(&wallets).Error; err != nil {
		return nil, fmt.Errorf("failed to get wallets by status: %w", err)
	}
	return wallets, nil
}

func (r *walletRepository) GetTotalBalance() (float64, error) {
	var total float64
	err := r.db.Model(&models.Wallet{}).Select("COALESCE(SUM(balance), 0)").Scan(&total).Error
	if err != nil {
		return 0, fmt.Errorf("failed to get total balance: %w", err)
	}
	return total, nil
}

func (r *walletRepository) GetActiveWalletsCount() (int64, error) {
	var count int64
	err := r.db.Model(&models.Wallet{}).Where("status = ?", "active").Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to get active wallets count: %w", err)
	}
	return count, nil
}

func (r *walletRepository) GetTransactionStats(start, end time.Time) (*TransactionStats, error) {
	var stats TransactionStats
	err := r.db.Model(&models.Transaction{}).
		Where("created_at BETWEEN ? AND ?", start, end).
		Select(`
			COUNT(*) as total_transactions,
			COALESCE(SUM(amount), 0) as total_volume,
			COALESCE(AVG(amount), 0) as avg_amount,
			COALESCE(MAX(amount), 0) as max_amount,
			COALESCE(MIN(amount), 0) as min_amount,
			COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) * 100.0 / COUNT(*), 0) as success_rate
		`).
		Scan(&stats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction stats: %w", err)
	}
	return &stats, nil
}
