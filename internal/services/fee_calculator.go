package services

type FeeCalculator struct{}

func NewFeeCalculator() *FeeCalculator {
	return &FeeCalculator{}
}

func (f *FeeCalculator) CalculateFee(amount float64) float64 {
	return amount * 0.01 // 1% fee
}
