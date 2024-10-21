package middleware

import (
	"landmark-api/internal/config"
	"landmark-api/internal/models"
	"landmark-api/internal/services"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type RateLimiter struct {
	config   *config.RateLimitConfig
	users    map[string]int
	ipLimits map[string]*IPLimit
	mu       sync.Mutex
	reset    map[string]time.Time
}

type IPLimit struct {
	count    int
	lastSeen time.Time
}

func NewRateLimiter(config *config.RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		config:   config,
		users:    make(map[string]int),
		ipLimits: make(map[string]*IPLimit),
		reset:    make(map[string]time.Time),
	}
}

func (rl *RateLimiter) GetLimit(plan models.SubscriptionPlan) int {
	return rl.config.Limits[plan]
}

func (rl *RateLimiter) RateLimit(authService services.AuthService, apiUsageService services.APIUsageService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				http.Error(w, "Invalid IP address", http.StatusBadRequest)
				return
			}

			if rl.isIPRateLimited(ip) {
				http.Error(w, "IP rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
				return
			}

			user, bl := services.UserFromContext(r.Context())
			if bl == true {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			subscription, bl := services.SubscriptionFromContext(r.Context())
			if bl == true {
				http.Error(w, "Subscription not found", http.StatusForbidden)
				return
			}

			usageStats, err := apiUsageService.GetCurrentUsage(r.Context(), user.ID, subscription.PlanType)
			if err != nil {
				http.Error(w, "Failed to get usage statistics", http.StatusInternalServerError)
				return
			}

			limit := rl.config.Limits[subscription.PlanType]
			if limit >= 0 && usageStats.CurrentCount >= limit {
				rl.setRateLimitHeaders(w, limit, 0, usageStats.PeriodEnd)
				http.Error(w, "Rate limit exceeded. Please upgrade your subscription for higher limits.", http.StatusTooManyRequests)
				return
			}

			wrappedWriter := &responseWriterWrapper{ResponseWriter: w}
			next.ServeHTTP(wrappedWriter, r)

			isCacheHit := wrappedWriter.Header().Get("X-Cache") == "HIT"

			if !isCacheHit {
				if err := apiUsageService.IncrementUsage(user.ID); err != nil {
					// Log the error, but don't fail the request
					println("Error incrementing usage:", err.Error())
				}
				usageStats.CurrentCount++
			}

			rl.setRateLimitHeaders(w, limit, limit-usageStats.CurrentCount, usageStats.PeriodEnd)
		})
	}
}

func (rl *RateLimiter) isIPRateLimited(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	limit, exists := rl.ipLimits[ip]

	if !exists {
		rl.ipLimits[ip] = &IPLimit{count: 1, lastSeen: now}
		return false
	}

	if now.Sub(limit.lastSeen) > time.Minute {
		limit.count = 1
		limit.lastSeen = now
		return false
	}

	limit.count++
	limit.lastSeen = now

	return limit.count > rl.config.IPBurstLimit
}

func (rl *RateLimiter) setRateLimitHeaders(w http.ResponseWriter, limit, remaining int, reset time.Time) {
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))
}

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
