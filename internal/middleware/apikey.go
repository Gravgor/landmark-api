package middleware

import (
	"landmark-api/internal/services"
	"net/http"

	"github.com/gorilla/mux"
)

// APIKeyMiddleware - Middleware to validate API keys and user ownership
func APIKeyMiddleware(apiKeyService services.APIKeyService) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("x-api-key")
			if apiKey == "" {
				http.Error(w, "API key is required", http.StatusUnauthorized)
				return
			}

			user, _ := services.UserFromContext(r.Context())
			if user == nil {
				http.Error(w, "User not authenticated", http.StatusUnauthorized)
				return
			}

			apiKeyDetails, err := apiKeyService.GetAPIKeyByKey(r.Context(), apiKey)
			if err != nil {
				http.Error(w, "Invalid API key", http.StatusUnauthorized)
				return
			}

			if apiKeyDetails.UserID != user.ID {
				http.Error(w, "API key does not belong to the user", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
