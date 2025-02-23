package wallet

import "time"

const (
	// Operation types
	OperationTypeCredit = "CREDIT"
	OperationTypeDebit  = "DEBIT"

	// Default values
	DefaultCurrency = "USD"

	// Cache durations
	BalanceCacheDuration = 5 * time.Minute
)
