package models

type UserType string

const (
	UserTypeRegular    UserType = "regular"
	UserTypeMerchant   UserType = "merchant"
	UserTypeEnterprise UserType = "enterprise"
)

type FeeStructure struct {
	WithdrawalFee        float64 `json:"withdrawal_fee"`
	TransactionFee       float64 `json:"transaction_fee"`
	MonthlyFee           float64 `json:"monthly_fee"`
	InstantWithdrawalFee float64 `json:"instant_withdrawal_fee"`
	MinimumBalance       float64 `json:"minimum_balance"`
}

var FeeStructures = map[UserType]FeeStructure{
	UserTypeRegular: {
		WithdrawalFee:        1.5, // 1.5% per withdrawal
		TransactionFee:       0,   // Free P2P transfers
		MonthlyFee:           0,   // No monthly fee
		InstantWithdrawalFee: 1.0, // +1% for instant withdrawal
		MinimumBalance:       0,
	},
	UserTypeMerchant: {
		WithdrawalFee:        1.0,  // 1% per withdrawal
		TransactionFee:       2.5,  // 2.5% per customer payment
		MonthlyFee:           9.99, // Monthly subscription
		InstantWithdrawalFee: 0.5,  // +0.5% for instant withdrawal
		MinimumBalance:       100,
	},
	UserTypeEnterprise: {
		WithdrawalFee:        0.5,   // 0.5% per withdrawal
		TransactionFee:       1.8,   // 1.8% per customer payment
		MonthlyFee:           29.99, // Monthly subscription
		InstantWithdrawalFee: 0,     // Free instant withdrawals
		MinimumBalance:       1000,
	},
}
