package middleware

import (
	"landmark-api/internal/services"
	"net/http"

	"github.com/gorilla/mux"
)

func APIKeyMiddleware(apiKeyService services.APIKeyService) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("x-api-key")
			if apiKey == "" {
				http.Error(w, "API key is required", http.StatusUnauthorized)
				return
			}

			user, subscription, err := apiKeyService.GetUserAndSubscriptionByAPIKey(r.Context(), apiKey)
			if err != nil {
				http.Error(w, "Invalid API key", http.StatusUnauthorized)
				return
			}

			// Add the user and subscription to the request context
			ctx := services.WithUserAndSubscriptionContext(r.Context(), user, subscription)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
