package transaction

// Default configuration values
const (
	DefaultMaxRetries     = 3
	DefaultProcessingTime = 30 // seconds
	DefaultMaxAmount      = 10000.0
	DefaultMinAmount      = 1.0
)

// Transaction processing modes
const (
	ProcessingModeSync  = "sync"
	ProcessingModeAsync = "async"
)

// Cache keys
const (
	TransactionCachePrefix = "transaction:"
	StatusCachePrefix      = "transaction_status:"
)
