package repositories

import (
	"orus/internal/models"
	"time"
)

func CreateQRCode(qr *models.QRCode) (*models.QRCode, error) {
	if err := DB.Create(qr).Error; err != nil {
		return nil, err
	}
	return qr, nil
}

func GetQRCodeByCode(code string) (*models.QRCode, error) {
	var qr models.QRCode
	if err := DB.Where("code = ?", code).First(&qr).Error; err != nil {
		return nil, err
	}
	return &qr, nil
}

func GetQRCodeByCodeForUpdate(code string) (*models.QRCode, error) {
	var qr models.QRCode
	if err := DB.Set("gorm:for_update", true).Where("code = ?", code).First(&qr).Error; err != nil {
		return nil, err
	}
	return &qr, nil
}

func GetQRCodeDailyTotal(qrID uint) (float64, error) {
	var total float64
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	err := DB.Model(&models.QRTransaction{}).
		Where("qr_code_id = ? AND created_at >= ? AND created_at < ?", qrID, today, tomorrow).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error

	return total, err
}

func GetQRCodeMonthlyTotal(qrID uint) (float64, error) {
	var total float64
	startOfMonth := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -time.Now().UTC().Day()+1)
	startOfNextMonth := startOfMonth.AddDate(0, 1, 0)

	err := DB.Model(&models.QRTransaction{}).
		Where("qr_code_id = ? AND created_at >= ? AND created_at < ?", qrID, startOfMonth, startOfNextMonth).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total).Error

	return total, err
}
