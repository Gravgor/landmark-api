// middleware/rate_limiter.go
package middleware

import (
	"landmark-api/internal/config"
	"landmark-api/internal/models"
	"landmark-api/internal/services"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type RateLimiter struct {
	config *config.RateLimitConfig
	users  map[string]int
	mu     sync.Mutex
	reset  map[string]time.Time
}

func NewRateLimiter(config *config.RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		config: config,
		users:  make(map[string]int),
		reset:  make(map[string]time.Time),
	}
}

func (rl *RateLimiter) GetLimit(plan models.SubscriptionPlan) int {
	return rl.config.Limits[plan]
}

func (rl *RateLimiter) RateLimit(authService services.AuthService, apiUsageService services.APIUsageService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := services.UserFromContext(r.Context())
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			subscription, ok := services.SubscriptionFromContext(r.Context())
			if !ok {
				http.Error(w, "Subscription not found", http.StatusForbidden)
				return
			}

			// Get usage from database
			usageStats, err := apiUsageService.GetCurrentUsage(user.ID.String(), subscription.PlanType)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// Check if user has exceeded their limit
			limit := rl.config.Limits[subscription.PlanType] // Fixed: Changed limits to Limits
			if limit >= 0 && usageStats.CurrentCount >= limit {
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(usageStats.PeriodEnd.Unix(), 10))
				http.Error(w, "Rate limit exceeded. Please upgrade your subscription for higher limits.", http.StatusTooManyRequests)
				return
			}

			// Increment usage
			if err := apiUsageService.IncrementUsage(user.ID.String()); err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// Update headers with current usage
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(limit-usageStats.CurrentCount-1))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(usageStats.PeriodEnd.Unix(), 10))

			next.ServeHTTP(w, r)
		})
	}
}
