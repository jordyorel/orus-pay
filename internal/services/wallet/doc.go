/*
Package wallet provides wallet management functionality for the application.

The wallet service handles all wallet-related operations including:
- Balance management (credit/debit)
- Transaction history
- Limits enforcement (daily/monthly)
- Batch operations
- Cache management

Usage:

	// Create a new wallet service
	svc := wallet.NewService(db, cache, config, metrics)

	// Create a new wallet
	wallet, err := svc.CreateWallet(ctx, userID, "USD")

	// Credit amount
	err = svc.Credit(ctx, userID, amount)

	// Debit amount
	err = svc.Debit(ctx, userID, amount)

	// Get transaction history
	history, err := svc.GetTransactionHistory(ctx, walletID, limit, offset)

Configuration:

The service can be configured using WalletConfig:

	config := wallet.WalletConfig{
	    DefaultCurrency:   "USD",
	    MaxDailyLimit:    10000.0,
	    MaxMonthlyLimit:  50000.0,
	    MinBalance:       0.0,
	    ProcessingTimeout: 30 * time.Second,
	}

Error Handling:

The service returns specific errors for different scenarios:
- ErrInvalidCurrency: When currency is not supported
- ErrDailyLimitExceeded: When daily transaction limit is exceeded
- ErrMonthlyLimitExceeded: When monthly transaction limit is exceeded
- ErrWalletLocked: When wallet is locked
- ErrInvalidOperation: For general invalid operations

Cache Management:

The service implements a caching strategy for:
- Wallet data
- Balance information
- Transaction history
- Daily/monthly totals

Metrics:

The service collects metrics for:
- Operation durations
- Cache hit/miss rates
- Transaction volumes
- Error rates
- Balance changes
*/
package wallet
