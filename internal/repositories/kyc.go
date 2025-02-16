package repositories

import "orus/internal/models"

func CreateKYC(kyc *models.KYCVerification) error {
	return DB.Create(kyc).Error
}

func GetKYCByDocumentID(docID string) (*models.KYCVerification, error) {
	var kyc models.KYCVerification
	err := DB.Where("document_id = ?", docID).First(&kyc).Error
	return &kyc, err
}
