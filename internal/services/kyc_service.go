package services

import (
	"context"
	"orus/internal/models"
	"orus/internal/repositories"
)

// KYCService defines verification operations.
type KYCService interface {
	SubmitKYC(ctx context.Context, userID uint, documentID, scanURL string) (*models.KYCVerification, error)
	GetStatus(ctx context.Context, userID uint) (*models.KYCVerification, error)
}

type kycService struct{}

// NewKYCService creates a new KYCService.
func NewKYCService() KYCService { return &kycService{} }

func (s *kycService) SubmitKYC(ctx context.Context, userID uint, documentID, scanURL string) (*models.KYCVerification, error) {
	kyc := &models.KYCVerification{
		UserID:     userID,
		DocumentID: documentID,
		ScanURL:    scanURL,
		Status:     "pending",
	}
	if err := repositories.CreateKYC(kyc); err != nil {
		return nil, err
	}
	return kyc, nil
}

func (s *kycService) GetStatus(ctx context.Context, userID uint) (*models.KYCVerification, error) {
	var kyc models.KYCVerification
	if err := repositories.DB.Where("user_id = ?", userID).First(&kyc).Error; err != nil {
		return nil, err
	}
	return &kyc, nil
}
