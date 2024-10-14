// internal/handlers/usage_handler.go
package handlers

import (
	"encoding/json"
	"landmark-api/internal/services"
	"net/http"
)

type UsageHandler struct {
	usageService services.APIUsageService
	authService  services.AuthService
}

func NewUsageHandler(usageService services.APIUsageService, authService services.AuthService) *UsageHandler {
	return &UsageHandler{
		usageService: usageService,
		authService:  authService,
	}
}

func (h *UsageHandler) GetCurrentUsage(w http.ResponseWriter, r *http.Request) {
	user, ok := services.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	subscription, err := h.authService.GetCurrentSubscription(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Error fetching subscription", http.StatusInternalServerError)
		return
	}

	stats, err := h.usageService.GetCurrentUsage(r.Context(), user.ID, subscription.PlanType)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
