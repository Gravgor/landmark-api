package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"landmark-api/internal/models"
	"landmark-api/internal/services"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type SuggestionsHandler struct {
	db           *gorm.DB
	cacheService services.CacheService
}

// SuggestionResponse represents the structure expected by the frontend
type SuggestionResponse struct {
	Names      []string `json:"names"`
	Countries  []string `json:"countries"`
	Categories []string `json:"categories"`
}

func NewSuggestionsHandler(db *gorm.DB, cacheService services.CacheService) *SuggestionsHandler {
	return &SuggestionsHandler{
		db:           db,
		cacheService: cacheService,
	}
}

// GetSuggestions handles all types of suggestions (names, countries, categories)
func (h *SuggestionsHandler) GetSuggestions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	searchType := vars["type"] // will be "names", "countries", or "categories"
	searchTerm := r.URL.Query().Get("search")

	if searchTerm == "" {
		respondWithJSON(w, http.StatusOK, SuggestionResponse{
			Names:      []string{},
			Countries:  []string{},
			Categories: []string{},
		})
		return
	}

	// Get subscription from context
	subscription, ok := services.SubscriptionFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusForbidden, "Subscription not found")
		return
	}

	cacheKey := h.getCacheKey(fmt.Sprintf("suggestions:%s", searchType),
		searchTerm,
		string(subscription.PlanType))

	// Try to get from cache first
	if cachedData, err := h.cacheService.Get(ctx, cacheKey); err == nil {
		var response SuggestionResponse
		if err := json.Unmarshal([]byte(cachedData), &response); err == nil {
			w.Header().Set("X-Cache", "HIT")
			respondWithJSON(w, http.StatusOK, response)
			return
		}
	}

	response, err := h.fetchSuggestions(ctx, searchType, searchTerm, subscription)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error fetching suggestions")
		return
	}

	// Cache the response
	h.cacheService.Set(ctx, cacheKey, response, 5*time.Minute) // shorter cache time for suggestions
	w.Header().Set("X-Cache", "MISS")
	respondWithJSON(w, http.StatusOK, response)
}

func (h *SuggestionsHandler) fetchSuggestions(ctx context.Context, searchType, searchTerm string, subscription *models.Subscription) (SuggestionResponse, error) {
	var response SuggestionResponse
	limit := 10 // Limit suggestions to 10 items

	switch searchType {
	case "names":
		var names []string
		err := h.db.Model(&models.Landmark{}).
			Select("DISTINCT name").
			Where("name ILIKE ?", "%"+searchTerm+"%").
			Limit(limit).
			Pluck("name", &names).Error
		if err != nil {
			return response, err
		}
		response.Names = names

	case "countries":
		var countries []string
		err := h.db.Model(&models.Landmark{}).
			Select("DISTINCT country").
			Where("country ILIKE ?", "%"+searchTerm+"%").
			Limit(limit).
			Pluck("country", &countries).Error
		if err != nil {
			return response, err
		}
		response.Countries = countries

	case "categories":
		var categories []string
		err := h.db.Model(&models.Landmark{}).
			Select("DISTINCT category").
			Where("category ILIKE ?", "%"+searchTerm+"%").
			Limit(limit).
			Pluck("category", &categories).Error
		if err != nil {
			return response, err
		}
		response.Categories = categories
	}

	return response, nil
}

func (h *SuggestionsHandler) getCacheKey(parts ...string) string {
	key := "suggestions"
	for _, part := range parts {
		key += ":" + part
	}
	return key
}
