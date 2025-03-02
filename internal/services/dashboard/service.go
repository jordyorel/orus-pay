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
		Where("receiver_id = ? AND status = ?", merchantID, "completed").
		Select("COUNT(*) as total_transactions, COALESCE(SUM(amount), 0) as total_amount").
		Row().Scan(&dashboard.TotalTransactions, &dashboard.TotalAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to get total stats: %w", err)
	}

	// Get daily stats
	err = s.db.Model(&models.Transaction{}).
		Where("receiver_id = ? AND status = ? AND updated_at >= ?",
			merchantID, "completed", startOfDay).
		Select("COUNT(*) as daily_transactions, COALESCE(SUM(amount), 0) as daily_amount").
		Row().Scan(&dashboard.DailyTransactions, &dashboard.DailyAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily stats: %w", err)
	}

	// Get monthly stats
	err = s.db.Model(&models.Transaction{}).
		Where("receiver_id = ? AND status = ? AND updated_at >= ?",
			merchantID, "completed", startOfMonth).
		Select("COUNT(*) as monthly_transactions, COALESCE(SUM(amount), 0) as monthly_amount").
		Row().Scan(&dashboard.MonthlyTransactions, &dashboard.MonthlyAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly stats: %w", err)
	}

	// Get recent transactions
	err = s.db.Where("receiver_id = ? AND status = ?", merchantID, "completed").
		Order("updated_at DESC").
		Limit(10).
		Find(&dashboard.RecentTransactions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get recent transactions: %w", err)
	}

	return &dashboard, nil
}

func (s *service) getBasicStats(_ context.Context, userID uint) (*models.DashboardStats, error) {
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
		LastTransactionDate:      &lastTx.ProcessedAt,
		CurrentBalance:           wallet.Balance,
		PendingTransactions:      0, // TODO: Implement pending transactions count
	}, nil
}

func (s *service) GetTransactionAnalytics(ctx context.Context, userID uint, startDate, endDate time.Time) (map[string]interface{}, error) {
	// Check if user is a merchant
	merchant, err := s.merchantRepo.GetByUserID(userID)
	if err == nil && merchant != nil {
		// This is a merchant, get merchant-specific analytics
		fmt.Printf("Getting merchant analytics for userID %d (merchantID %d)\n", userID, merchant.ID)
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

	// Debug query parameters
	fmt.Printf("Query parameters - MerchantID: %d, Status: completed, Start: %v, End: %v\n",
		merchantID, startDate, endDate)

	// First, let's check if we can find any transactions at all for this merchant
	var totalTx int64
	err := s.db.Model(&models.Transaction{}).
		Where("merchant_id = ?", merchantID).
		Count(&totalTx).Error
	if err != nil {
		return nil, fmt.Errorf("failed to count total transactions: %w", err)
	}
	fmt.Printf("Total transactions found for merchant (without filters): %d\n", totalTx)

	// Get daily volumes with more detailed logging
	volumeOverTime := make(map[string]float64)
	var dailyStats []struct {
		Date   string
		Count  int
		Volume float64
	}

	err = s.db.Model(&models.Transaction{}).
		Select("DATE(processed_at)::text as date, COUNT(*) as count, COALESCE(SUM(amount), 0) as volume").
		Where("merchant_id = ? AND status = ? AND processed_at >= ? AND processed_at <= ?",
			merchantID, "completed", startDate, endDate).
		Group("DATE(processed_at)").
		Order("date").
		Find(&dailyStats).Error

	if err != nil {
		fmt.Printf("Error getting daily volumes: %v\n", err)
		return nil, fmt.Errorf("failed to get daily volumes: %w", err)
	}

	for _, stat := range dailyStats {
		volumeOverTime[stat.Date] = stat.Volume
		fmt.Printf("Daily stat - Date: %s, Count: %d, Volume: %.2f\n",
			stat.Date, stat.Count, stat.Volume)
	}

	// Get payment method counts with debugging
	countByType := make(map[string]int64)
	var methodStats []struct {
		Method string
		Count  int64
	}

	err = s.db.Model(&models.Transaction{}).
		Select("COALESCE(payment_method, 'unknown') as method, COUNT(*) as count").
		Where("merchant_id = ? AND status = ? AND processed_at >= ? AND processed_at <= ?",
			merchantID, "completed", startDate, endDate).
		Group("payment_method").
		Find(&methodStats).Error

	if err != nil {
		fmt.Printf("Error getting payment method counts: %v\n", err)
		return nil, fmt.Errorf("failed to get payment method counts: %w", err)
	}

	for _, stat := range methodStats {
		countByType[stat.Method] = stat.Count
		fmt.Printf("Payment method stat - Method: %s, Count: %d\n",
			stat.Method, stat.Count)
	}

	// Get summary stats with more detailed query
	var summary struct {
		TotalVolume        float64
		AverageTransaction float64
		TotalCount         int64
	}

	err = s.db.Model(&models.Transaction{}).
		Where("merchant_id = ? AND status = ? AND processed_at >= ? AND processed_at <= ?",
			merchantID, "completed", startDate, endDate).
		Select(`
			COALESCE(SUM(amount), 0) as total_volume,
			CASE 
				WHEN COUNT(*) > 0 THEN COALESCE(SUM(amount), 0) / COUNT(*)
				ELSE 0
			END as average_transaction,
			COUNT(*) as total_count
		`).
		Scan(&summary).Error

	if err != nil {
		fmt.Printf("Error getting summary stats: %v\n", err)
		return nil, fmt.Errorf("failed to get summary stats: %w", err)
	}

	// Debug summary results
	fmt.Printf("Summary stats for merchant %d:\n", merchantID)
	fmt.Printf("- Total volume: %.2f\n", summary.TotalVolume)
	fmt.Printf("- Average transaction: %.2f\n", summary.AverageTransaction)
	fmt.Printf("- Total count: %d\n", summary.TotalCount)
	fmt.Printf("- Volume over time entries: %d\n", len(volumeOverTime))
	fmt.Printf("- Payment methods found: %d\n", len(countByType))

	// Check if we have any data
	if summary.TotalCount == 0 {
		fmt.Printf("WARNING: No transactions found for merchant %d in date range\n", merchantID)
		// Double check the date range
		fmt.Printf("Date range check - Start: %v, End: %v\n", startDate, endDate)
	}

	return map[string]interface{}{
		"volume_over_time": volumeOverTime,
		"count_by_type":    countByType,
		"summary": map[string]interface{}{
			"total_volume":        summary.TotalVolume,
			"average_transaction": summary.AverageTransaction,
			"total_count":         summary.TotalCount,
		},
	}, nil
}
