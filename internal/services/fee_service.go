package services

import (
	"orus/internal/models"
)

func (fc *FeeCalculator) CalculateWithdrawalFee(amount float64, userType models.UserType, instant bool) float64 {
	feeStructure := models.FeeStructures[userType]
	fee := amount * (feeStructure.WithdrawalFee / 100)

	if instant {
		fee += amount * (feeStructure.InstantWithdrawalFee / 100)
	}

	return fee
}

func (fc *FeeCalculator) CalculateTransactionFee(amount float64, userType models.UserType) float64 {
	feeStructure := models.FeeStructures[userType]
	return amount * (feeStructure.TransactionFee / 100)
}

func (fc *FeeCalculator) GetMonthlyFee(userType models.UserType) float64 {
	return models.FeeStructures[userType].MonthlyFee
}

func (fc *FeeCalculator) ValidateMinimumBalance(balance float64, userType models.UserType) bool {
	return balance >= models.FeeStructures[userType].MinimumBalance
}
