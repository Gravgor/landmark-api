package repository

import (
	"landmark-api/internal/models"
	"time"

	"gorm.io/gorm"
)

type APIUsageRepository interface {
	GetCurrentUsage(userID string, periodStart, periodEnd time.Time) (*models.APIUsage, error)
	IncrementUsage(userID string) error
	CreateNewPeriod(usage *models.APIUsage) error
}

type apiUsageRepository struct {
	db *gorm.DB
}

func NewAPIUsageRepository(db *gorm.DB) APIUsageRepository {
	return &apiUsageRepository{db: db}
}

func (r *apiUsageRepository) GetCurrentUsage(userID string, periodStart, periodEnd time.Time) (*models.APIUsage, error) {
	var usage models.APIUsage
	err := r.db.Where("user_id = ? AND period_start = ? AND period_end = ?",
		userID, periodStart, periodEnd).First(&usage).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &usage, err
}

func (r *apiUsageRepository) IncrementUsage(userID string) error {
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Second)

	return r.db.Transaction(func(tx *gorm.DB) error {
		var usage models.APIUsage
		err := tx.Where("user_id = ? AND period_start = ? AND period_end = ?",
			userID, periodStart, periodEnd).First(&usage).Error

		if err == gorm.ErrRecordNotFound {
			usage = models.APIUsage{
				UserID:       userID,
				RequestCount: 1,
				PeriodStart:  periodStart,
				PeriodEnd:    periodEnd,
			}
			return tx.Create(&usage).Error
		}

		if err != nil {
			return err
		}

		usage.RequestCount++
		return tx.Save(&usage).Error
	})
}

func (r *apiUsageRepository) CreateNewPeriod(usage *models.APIUsage) error {
	return r.db.Create(usage).Error
}
