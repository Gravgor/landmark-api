package services

import (
	"context"
	"landmark-api/internal/config"
	"landmark-api/internal/models"
	"landmark-api/internal/repository"
	"time"

	"github.com/google/uuid"
)

type APIUsageService interface {
	GetCurrentUsage(ctx context.Context, userID uuid.UUID, plan models.SubscriptionPlan) (*UsageStats, error)
	IncrementUsage(userID uuid.UUID) error
}

type UsageStats struct {
	CurrentCount      int
	Limit             int
	RemainingRequests int
	PeriodEnd         time.Time
}

type apiUsageService struct {
	repo       repository.APIUsageRepository
	subRepo    repository.SubscriptionRepository
	rateConfig *config.RateLimitConfig
}

func NewAPIUsageService(repo repository.APIUsageRepository, subRepo repository.SubscriptionRepository, rateConfig *config.RateLimitConfig) APIUsageService {
	return &apiUsageService{
		repo:       repo,
		subRepo:    subRepo,
		rateConfig: rateConfig,
	}
}

func (s *apiUsageService) GetCurrentUsage(ctx context.Context, userID uuid.UUID, plan models.SubscriptionPlan) (*UsageStats, error) {
	// Fetch the user's subscription details
	subscription, err := s.subRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return nil, err
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
	}

	usage, err := s.repo.GetCurrentUsage(userID.String(), periodStart, periodEnd)
	if err != nil {
		return nil, err
	}

	if usage == nil {
		usage = &models.APIUsage{
			UserID:       userID.String(),
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

func (s *apiUsageService) IncrementUsage(userID uuid.UUID) error {
	return s.repo.IncrementUsage(userID)
}
