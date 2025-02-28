package dashboard

import (
	"context"
	"fmt"
	"orus/internal/models"
	"orus/internal/repositories"
	"time"

	"gorm.io/gorm"
)

type Service interface {
	GetUserDashboard(ctx context.Context, userID uint) (*models.UserDashboardStats, error)
	GetMerchantDashboard(ctx context.Context, merchantID uint) (*MerchantDashboard, error)
	GetTransactionAnalytics(ctx context.Context, userID uint, startDate, endDate time.Time) (map[string]interface{}, error)
}

type service struct {
	transactionRepo repositories.TransactionRepository
	walletRepo      repositories.WalletRepository
	merchantRepo    repositories.MerchantRepository
	userRepo        repositories.UserRepository
	db              *gorm.DB
}

type MerchantDashboard struct {
	TotalTransactions   int64                `json:"total_transactions"`
	TotalAmount         float64              `json:"total_amount"`
	DailyTransactions   int64                `json:"daily_transactions"`
	DailyAmount         float64              `json:"daily_amount"`
	MonthlyTransactions int64                `json:"monthly_transactions"`
	MonthlyAmount       float64              `json:"monthly_amount"`
	RecentTransactions  []models.Transaction `json:"recent_transactions"`
}

func NewService(
	transactionRepo repositories.TransactionRepository,
	walletRepo repositories.WalletRepository,
	merchantRepo repositories.MerchantRepository,
	userRepo repositories.UserRepository,
	db *gorm.DB,
) Service {
	return &service{
		transactionRepo: transactionRepo,
		walletRepo:      walletRepo,
		merchantRepo:    merchantRepo,
		userRepo:        userRepo,
		db:              db,
	}
}

func (s *service) GetUserDashboard(ctx context.Context, userID uint) (*models.UserDashboardStats, error) {
	// Get basic stats
	stats, err := s.getBasicStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get recent transactions
	recentMerchants, err := s.transactionRepo.GetRecentMerchants(userID, 5)
	if err != nil {
		return nil, err
	}

	// Get spending by category
	spendingByCategory, err := s.transactionRepo.GetSpendingByCategory(userID, time.Now().AddDate(0, -1, 0))
	if err != nil {
		return nil, err
	}

	// Get income by category
	incomeByCategory, err := s.transactionRepo.GetIncomeByCategory(userID, time.Now().AddDate(0, -1, 0))
	if err != nil {
		return nil, err
	}

	return &models.UserDashboardStats{
		DashboardStats:     *stats,
		SavedCards:         0, // TODO: Implement saved cards count
		RecentMerchants:    recentMerchants,
		SpendingByCategory: spendingByCategory,
		IncomeByCategory:   incomeByCategory,
		MonthlySpending:    stats.TotalVolume,
	}, nil
}

func (s *service) GetMerchantDashboard(ctx context.Context, merchantID uint) (*MerchantDashboard, error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	var dashboard MerchantDashboard

	// Get total stats
	err := s.db.Model(&models.Transaction{}).
		Where("merchant_id = ? AND status = ?", merchantID, "completed").
		Select("COUNT(*) as total_transactions, COALESCE(SUM(amount), 0) as total_amount").
		Row().Scan(&dashboard.TotalTransactions, &dashboard.TotalAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to get total stats: %w", err)
	}

	// Get daily stats
	err = s.db.Model(&models.Transaction{}).
		Where("merchant_id = ? AND status = ? AND created_at >= ?",
			merchantID, "completed", startOfDay).
		Select("COUNT(*) as daily_transactions, COALESCE(SUM(amount), 0) as daily_amount").
		Row().Scan(&dashboard.DailyTransactions, &dashboard.DailyAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily stats: %w", err)
	}

	// Get monthly stats
	err = s.db.Model(&models.Transaction{}).
		Where("merchant_id = ? AND status = ? AND created_at >= ?",
			merchantID, "completed", startOfMonth).
		Select("COUNT(*) as monthly_transactions, COALESCE(SUM(amount), 0) as monthly_amount").
		Row().Scan(&dashboard.MonthlyTransactions, &dashboard.MonthlyAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly stats: %w", err)
	}

	// Get recent transactions
	err = s.db.Where("merchant_id = ? AND status = ?", merchantID, "completed").
		Order("created_at DESC").
		Limit(10).
		Find(&dashboard.RecentTransactions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get recent transactions: %w", err)
	}

	return &dashboard, nil
}

func (s *service) getBasicStats(ctx context.Context, userID uint) (*models.DashboardStats, error) {
	// Get transaction count and volume
	count, volume, err := s.transactionRepo.GetTransactionStats(userID)
	if err != nil {
		return nil, err
	}

	// Get last transaction date
	lastTx, err := s.transactionRepo.GetLastTransaction(userID)
	if err != nil {
		return nil, err
	}

	// Get wallet balance
	wallet, err := s.walletRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}

	return &models.DashboardStats{
		TotalTransactions:        count,
		TotalVolume:              volume,
		AverageTransactionAmount: volume / float64(count),
		LastTransactionDate:      &lastTx.CreatedAt,
		CurrentBalance:           wallet.Balance,
		PendingTransactions:      0, // TODO: Implement pending transactions count
	}, nil
}

func (s *service) GetTransactionAnalytics(ctx context.Context, userID uint, startDate, endDate time.Time) (map[string]interface{}, error) {
	// Check if user is a merchant
	merchant, err := s.merchantRepo.GetByUserID(userID)
	fmt.Printf("Looking up merchant for userID %d: %v, err: %v\n", userID, merchant, err)
	if err == nil && merchant != nil {
		// This is a merchant, get merchant-specific analytics
		fmt.Printf("Getting merchant analytics for merchantID %d\n", merchant.ID)
		return s.getMerchantAnalytics(merchant.ID, startDate, endDate)
	}

	// Regular user analytics
	volumeOverTime, err := s.transactionRepo.GetVolumeOverTime(userID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	countByType, err := s.transactionRepo.GetTransactionCountByType(userID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"volume_over_time": volumeOverTime,
		"count_by_type":    countByType,
	}, nil
}

func (s *service) getMerchantAnalytics(merchantID uint, startDate, endDate time.Time) (map[string]interface{}, error) {
	fmt.Printf("Querying transactions for merchantID %d between %v and %v\n", merchantID, startDate, endDate)

	// Get daily transaction volumes
	dailyVolumes, err := s.db.Model(&models.Transaction{}).
		Select("DATE(created_at) as date, COUNT(*) as count, SUM(amount) as volume").
		Where("merchant_id = ? AND status = ? AND created_at BETWEEN ? AND ?",
			merchantID, "completed", startDate, endDate).
		Group("DATE(created_at)").
		Order("date").
		Rows()
	if err != nil {
		return nil, fmt.Errorf("failed to get daily volumes: %w", err)
	}
	defer dailyVolumes.Close()

	volumeOverTime := make(map[string]float64)
	for dailyVolumes.Next() {
		var date string
		var count int
		var volume float64
		if err := dailyVolumes.Scan(&date, &count, &volume); err != nil {
			return nil, err
		}
		volumeOverTime[date] = volume
	}

	// Get transaction counts by payment method
	var countByType = make(map[string]int64)
	rows, err := s.db.Model(&models.Transaction{}).
		Select("payment_method, COUNT(*) as count").
		Where("merchant_id = ? AND status = ? AND created_at BETWEEN ? AND ?",
			merchantID, "completed", startDate, endDate).
		Group("payment_method").
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var paymentMethod string
		var count int64
		if err := rows.Scan(&paymentMethod, &count); err != nil {
			return nil, err
		}
		countByType[paymentMethod] = count
	}

	// Get summary stats
	var totalVolume, avgTransaction float64

	err = s.db.Model(&models.Transaction{}).
		Where("merchant_id = ? AND status = ? AND created_at BETWEEN ? AND ?",
			merchantID, "completed", startDate, endDate).
		Select("COALESCE(SUM(amount), 0)").
		Row().Scan(&totalVolume)
	if err != nil {
		return nil, fmt.Errorf("failed to get total volume: %w", err)
	}

	err = s.db.Model(&models.Transaction{}).
		Where("merchant_id = ? AND status = ? AND created_at BETWEEN ? AND ?",
			merchantID, "completed", startDate, endDate).
		Select("COALESCE(AVG(amount), 0)").
		Row().Scan(&avgTransaction)
	if err != nil {
		return nil, fmt.Errorf("failed to get average transaction: %w", err)
	}

	// Debug: Check if we have any transactions at all
	var totalCount int64
	s.db.Model(&models.Transaction{}).
		Where("merchant_id = ?", merchantID).
		Count(&totalCount)
	fmt.Printf("Total transactions found for merchantID %d: %d\n", merchantID, totalCount)

	return map[string]interface{}{
		"volume_over_time": volumeOverTime,
		"count_by_type":    countByType,
		"summary": map[string]interface{}{
			"total_volume":        totalVolume,
			"average_transaction": avgTransaction,
		},
	}, nil
}
