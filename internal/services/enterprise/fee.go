package enterprise

import (
	"orus/internal/models"
)

type FeeCalculator struct {
	baseFee     float64
	percentRate float64
}

func NewFeeCalculator() *FeeCalculator {
	return &FeeCalculator{
		baseFee:     0.30,  // Base fee in currency units
		percentRate: 0.029, // 2.9% standard rate
	}
}

func (fc *FeeCalculator) CalculateTransactionFee(amount float64, userType models.UserType) float64 {
	return fc.baseFee + (amount * fc.percentRate)
}
