package repository

import (
	"landmark-api/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type APIUsageRepository interface {
	GetCurrentUsage(userID string, periodStart, periodEnd time.Time) (*models.APIUsage, error)
	IncrementUsage(userID uuid.UUID) error
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

func (r *apiUsageRepository) IncrementUsage(userID uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Fetch the user's subscription
		var subscription models.Subscription
		if err := tx.Where("user_id = ?", userID).First(&subscription).Error; err != nil {
			return err
		}

		now := time.Now()
		periodStart := subscription.StartDate
		periodEnd := subscription.EndDate

		// If the current time is past the billing date, adjust the period
		if now.After(periodEnd) {
			for periodEnd.Before(now) {
				periodStart = periodEnd
				periodEnd = periodEnd.AddDate(0, 1, 0)
			}
			// Update the subscription with the new billing date
			subscription.StartDate = periodStart
			subscription.EndDate = periodEnd
			if err := tx.Save(&subscription).Error; err != nil {
				return err
			}
		}

		// Find or create the API usage record for the current period
		var usage models.APIUsage
		err := tx.Where("user_id = ? AND period_start = ? AND period_end = ?",
			userID, periodStart, periodEnd).First(&usage).Error

		if err == gorm.ErrRecordNotFound {
			usage = models.APIUsage{
				UserID:       userID.String(),
				RequestCount: 1,
				PeriodStart:  periodStart,
				PeriodEnd:    periodEnd,
			}
			return tx.Create(&usage).Error
		}

		if err != nil {
			return err
		}

		// Increment the usage count
		usage.RequestCount++
		return tx.Save(&usage).Error
	})
}

func (r *apiUsageRepository) CreateNewPeriod(usage *models.APIUsage) error {
	return r.db.Create(usage).Error
}
