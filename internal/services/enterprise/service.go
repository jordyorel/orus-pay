package enterprise

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"orus/internal/models"
	"orus/internal/repositories"
	"time"
)

type EnterpriseService struct {
	feeCalculator *FeeCalculator
}

func NewEnterpriseService() *EnterpriseService {
	return &EnterpriseService{
		feeCalculator: NewFeeCalculator(),
	}
}

func (s *EnterpriseService) CreateEnterprise(enterprise *models.Enterprise) error {
	// Set default contract dates
	enterprise.ContractStartDate = time.Now()
	enterprise.ContractEndDate = time.Now().AddDate(1, 0, 0) // 1 year contract

	// Calculate sample transaction fee for 1000 units
	sampleFee := s.feeCalculator.CalculateTransactionFee(1000, models.UserTypeEnterprise)

	// Set default pricing plan
	defaultPricing, err := json.Marshal(map[string]interface{}{
		"transaction_fee_rate": (sampleFee / 1000) * 100, // Convert to percentage
		"monthly_fee":          299.99,
		"setup_fee":            999.99,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal pricing plan: %w", err)
	}

	enterprise.CustomPricingPlan = defaultPricing
	return repositories.DB.Create(enterprise).Error
}

func (s *EnterpriseService) AddLocation(enterpriseID uint, location models.EnterpriseLocation) error {
	location.EnterpriseID = enterpriseID
	return repositories.DB.Create(&location).Error
}

func (s *EnterpriseService) GenerateAPIKey(enterpriseID uint, keyName, environment string) (*models.EnterpriseAPIKey, error) {
	apiKey := &models.EnterpriseAPIKey{
		EnterpriseID: enterpriseID,
		KeyName:      keyName,
		Environment:  environment,
		APIKey:       generateSecureAPIKey(),
		Status:       "active",
	}

	if err := repositories.DB.Create(apiKey).Error; err != nil {
		return nil, err
	}

	return apiKey, nil
}

func (s *EnterpriseService) UpdateComplianceInfo(enterpriseID uint, officer, email string) error {
	updates := map[string]interface{}{
		"compliance_officer": officer,
		"compliance_email":   email,
		"last_audit_date":    time.Now(),
	}

	return repositories.DB.Model(&models.Enterprise{}).Where("id = ?", enterpriseID).
		Updates(updates).Error
}

func generateSecureAPIKey() string {
	// Implement secure API key generation
	return fmt.Sprintf("ent_%d_%s", time.Now().Unix(), generateRandomString(32))
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
