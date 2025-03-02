package repositories

import (
	"orus/internal/models"

	"gorm.io/gorm"
)

type DisputeRepository interface {
	Create(dispute *models.Dispute) error
	FindByID(id uint) (*models.Dispute, error)
	FindByMerchantID(merchantID uint) ([]models.Dispute, error)
	ExistsByTransactionID(transactionID uint) (bool, error)
	IsRefunded(disputeID uint) (bool, error)
	Update(dispute *models.Dispute) error
}

type disputeRepository struct {
	db *gorm.DB
}

func NewDisputeRepository(db *gorm.DB) DisputeRepository {
	return &disputeRepository{db: db}
}

func (r *disputeRepository) Create(dispute *models.Dispute) error {
	return r.db.Create(dispute).Error
}

func (r *disputeRepository) FindByID(id uint) (*models.Dispute, error) {
	var dispute models.Dispute
	err := r.db.First(&dispute, id).Error
	return &dispute, err
}

func (r *disputeRepository) FindByMerchantID(merchantID uint) ([]models.Dispute, error) {
	var disputes []models.Dispute
	err := r.db.Where("merchant_id = ?", merchantID).Find(&disputes).Error
	return disputes, err
}

func (r *disputeRepository) ExistsByTransactionID(transactionID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.Dispute{}).Where("transaction_id = ?", transactionID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *disputeRepository) IsRefunded(disputeID uint) (bool, error) {
	var dispute models.Dispute
	err := r.db.First(&dispute, disputeID).Error
	if err != nil {
		return false, err
	}
	return dispute.Refunded, nil
}

func (r *disputeRepository) Update(dispute *models.Dispute) error {
	return r.db.Save(dispute).Error
}
