package wallet

import "time"

// Wallet statuses
const (
	StatusActive   = "active"
	StatusLocked   = "locked"
	StatusInactive = "inactive"
)

// Default configuration values
const (
	DefaultCurrency        = "USD"
	DefaultMaxDailyLimit   = 10000.0
	DefaultMaxMonthlyLimit = 50000.0
	DefaultMinBalance      = 0.0
	DefaultTimeout         = 30 * time.Second
)

// Cache keys and durations
const (
	WalletCachePrefix = "wallet:"
	CacheDuration     = 5 * time.Minute
)
