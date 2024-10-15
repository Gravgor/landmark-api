package config

import (
	"landmark-api/internal/models"
)

type RateLimitConfig struct {
	Limits map[models.SubscriptionPlan]int
}

func NewRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Limits: map[models.SubscriptionPlan]int{
			models.FreePlan:       1000,
			models.ProPlan:        300000,
			models.EnterprisePlan: -1, // No limit for Enterprise
		},
	}
}
