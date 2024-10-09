package middleware

import (
	"landmark-api/internal/services"
	"net/http"

	"github.com/gorilla/mux"
)

// APIKeyMiddleware - Middleware to validate API keys
func APIKeyMiddleware(apiKeyService services.APIKeyService) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("x-api-key")
			if apiKey == "" {
				http.Error(w, "API key is required", http.StatusUnauthorized)
				return
			}

			_, err := apiKeyService.GetAPIKeyByKey(r.Context(), apiKey)
			if err != nil {
				http.Error(w, "Invalid API key", http.StatusUnauthorized)
				return
			}

			// If the API key is valid, call the next handler
			next.ServeHTTP(w, r)
		})
	}
}
