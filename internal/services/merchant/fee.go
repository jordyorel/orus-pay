package merchant

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

func (fc *FeeCalculator) CalculateFee(amount float64) float64 {
	return fc.baseFee + (amount * fc.percentRate)
}
