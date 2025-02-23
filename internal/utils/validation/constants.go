package validation

const (
	// Amount limits
	MinTransactionAmount = 0.01
	MaxTransactionAmount = 10000.00

	// Transaction limits
	MaxDailyTransactions   = 100
	MaxMonthlyTransactions = 1000

	// Password requirements
	MinPasswordLength = 8
	MaxPasswordLength = 72

	// String lengths
	MaxDescriptionLength = 500
	MaxReferenceLength   = 100
)
