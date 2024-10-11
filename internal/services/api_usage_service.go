package services

import (
	"landmark-api/internal/config"
	"landmark-api/internal/models"
	"landmark-api/internal/repository"
	"time"
)

type APIUsageService interface {
	GetCurrentUsage(userID string, plan models.SubscriptionPlan) (*UsageStats, error)
	IncrementUsage(userID string) error
}

type UsageStats struct {
	CurrentCount      int
	Limit             int
	RemainingRequests int
	PeriodEnd         time.Time
}

type apiUsageService struct {
	repo       repository.APIUsageRepository
	rateConfig *config.RateLimitConfig
}

func NewAPIUsageService(repo repository.APIUsageRepository, rateConfig *config.RateLimitConfig) APIUsageService {
	return &apiUsageService{
		repo:       repo,
		rateConfig: rateConfig,
	}
}

func (s *apiUsageService) GetCurrentUsage(userID string, plan models.SubscriptionPlan) (*UsageStats, error) {
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Second)

	usage, err := s.repo.GetCurrentUsage(userID, periodStart, periodEnd)
	if err != nil {
		return nil, err
	}

	if usage == nil {
		usage = &models.APIUsage{
			UserID:       userID,
			RequestCount: 0,
			PeriodStart:  periodStart,
			PeriodEnd:    periodEnd,
		}
	}

	limit := s.rateConfig.Limits[plan]

	return &UsageStats{
		CurrentCount:      usage.RequestCount,
		Limit:             limit,
		RemainingRequests: limit - usage.RequestCount,
		PeriodEnd:         periodEnd,
	}, nil
}

func (s *apiUsageService) IncrementUsage(userID string) error {
	return s.repo.IncrementUsage(userID)
}
