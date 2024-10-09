package middleware

import (
	"landmark-api/internal/models"
	"landmark-api/internal/services"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type RateLimiter struct {
	limits map[models.SubscriptionPlan]int
	users  map[string]int
	mu     sync.Mutex
	reset  map[string]time.Time
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limits: map[models.SubscriptionPlan]int{
			models.FreePlan:       1000,
			models.ProPlan:        5000,
			models.EnterprisePlan: -1, // No limit for Enterprise
		},
		users: make(map[string]int),
		reset: make(map[string]time.Time),
	}
}

func (rl *RateLimiter) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user and subscription from context
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

		rl.mu.Lock()
		defer rl.mu.Unlock()

		userID := user.ID.String()

		// Check if the user's limit should reset
		now := time.Now()
		if resetTime, exists := rl.reset[userID]; !exists || now.After(resetTime) {
			rl.reset[userID] = now.AddDate(0, 1, 0) // Reset in 30 days
			rl.users[userID] = 0
		}

		// Apply rate limiting based on subscription plan
		limit := rl.limits[subscription.PlanType]
		if limit >= 0 && rl.users[userID] >= limit {
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(rl.reset[userID].Unix(), 10))
			http.Error(w, "Rate limit exceeded. Please upgrade your subscription for higher limits.", http.StatusTooManyRequests)
			return
		}

		// Increment the user's request count
		rl.users[userID]++

		// Add rate limit headers
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(limit-rl.users[userID]))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(rl.reset[userID].Unix(), 10))

		next.ServeHTTP(w, r)
	})
}
