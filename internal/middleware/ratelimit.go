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
			usageStats, err := apiUsageService.GetCurrentUsage(r.Context(), user.ID, subscription.PlanType)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// Check if user has exceeded their limit
			limit := rl.config.Limits[subscription.PlanType]
			if limit >= 0 && usageStats.CurrentCount >= limit {
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(usageStats.PeriodEnd.Unix(), 10))
				http.Error(w, "Rate limit exceeded. Please upgrade your subscription for higher limits.", http.StatusTooManyRequests)
				return
			}

			// Wrap the ResponseWriter to capture the X-Cache header
			wrappedWriter := &responseWriterWrapper{ResponseWriter: w}

			// Call the next handler
			next.ServeHTTP(wrappedWriter, r)

			// Check if the response was served from cache
			isCacheHit := wrappedWriter.Header().Get("X-Cache") == "HIT"

			// Increment usage only if it's not a cache hit
			if !isCacheHit {
				if err := apiUsageService.IncrementUsage(user.ID.String()); err != nil {
					println("Error incrementing usage:", err.Error())
				}
				usageStats.CurrentCount++
			}

			// Update headers with current usage
			wrappedWriter.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			wrappedWriter.Header().Set("X-RateLimit-Remaining", strconv.Itoa(limit-usageStats.CurrentCount))
			wrappedWriter.Header().Set("X-RateLimit-Reset", strconv.FormatInt(usageStats.PeriodEnd.Unix(), 10))
		})
	}
}

// responseWriterWrapper is a simple wrapper around http.ResponseWriter that allows us to inspect headers after they've been written
type responseWriterWrapper struct {
	http.ResponseWriter
	wroteHeader bool
}

func (rww *responseWriterWrapper) WriteHeader(statusCode int) {
	rww.ResponseWriter.WriteHeader(statusCode)
	rww.wroteHeader = true
}

func (rww *responseWriterWrapper) Write(b []byte) (int, error) {
	if !rww.wroteHeader {
		rww.WriteHeader(http.StatusOK)
	}
	return rww.ResponseWriter.Write(b)
}
