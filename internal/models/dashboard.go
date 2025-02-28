package models

import "time"

// DashboardStats represents analytics data for user/merchant dashboards
type DashboardStats struct {
	TotalTransactions        int        `json:"total_transactions"`
	TotalVolume              float64    `json:"total_volume"`
	AverageTransactionAmount float64    `json:"average_transaction_amount"`
	LastTransactionDate      *time.Time `json:"last_transaction_date"`
	CurrentBalance           float64    `json:"current_balance"`
	PendingTransactions      int        `json:"pending_transactions"`
}

// MerchantDashboardStats extends DashboardStats with merchant-specific metrics
type MerchantDashboardStats struct {
	DashboardStats
	TotalCustomers int          `json:"total_customers"`
	MonthlyRevenue float64      `json:"monthly_revenue"`
	ProcessingFees float64      `json:"processing_fees"`
	ChargebackRate float64      `json:"chargeback_rate"`
	SuccessRate    float64      `json:"success_rate"`
	TopProducts    []TopProduct `json:"top_products"`
}

type TopProduct struct {
	Name             string  `json:"name"`
	TransactionCount int     `json:"transaction_count"`
	TotalRevenue     float64 `json:"total_revenue"`
}

// UserDashboardStats extends DashboardStats with user-specific metrics
type UserDashboardStats struct {
	DashboardStats
	SavedCards         int                `json:"saved_cards"`
	RecentMerchants    []string           `json:"recent_merchants"`
	SpendingByCategory map[string]float64 `json:"spending_by_category"`
	IncomeByCategory   map[string]float64 `json:"income_by_category"`
	MonthlySpending    float64            `json:"monthly_spending"`
}
