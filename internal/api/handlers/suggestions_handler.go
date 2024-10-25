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
	Cities     []string `json:"cities"`
}

func NewSuggestionsHandler(db *gorm.DB, cacheService services.CacheService) *SuggestionsHandler {
	return &SuggestionsHandler{
		db:           db,
		cacheService: cacheService,
	}
}

// GetSuggestions handles all types of suggestions (names, countries, categories, cities)
func (h *SuggestionsHandler) GetSuggestions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	searchType := vars["type"] // will be "names", "countries", "categories", or "cities"
	searchTerm := r.URL.Query().Get("search")

	if searchTerm == "" {
		respondWithJSON(w, http.StatusOK, SuggestionResponse{
			Names:      []string{},
			Countries:  []string{},
			Categories: []string{},
			Cities:     []string{},
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

	case "cities":
		var cities []string
		err := h.db.Model(&models.Landmark{}).
			Select("DISTINCT city").
			Where("city ILIKE ?", "%"+searchTerm+"%").
			Limit(limit).
			Pluck("city", &cities).Error
		if err != nil {
			return response, err
		}
		response.Cities = cities
	}

	return response, nil
}

// Optional: Add a method to get combined suggestions
func (h *SuggestionsHandler) GetCombinedSuggestions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	searchTerm := r.URL.Query().Get("search")

	if searchTerm == "" {
		respondWithJSON(w, http.StatusOK, SuggestionResponse{
			Names:      []string{},
			Countries:  []string{},
			Categories: []string{},
			Cities:     []string{},
		})
		return
	}

	subscription, ok := services.SubscriptionFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusForbidden, "Subscription not found")
		return
	}

	cacheKey := h.getCacheKey("suggestions:combined", searchTerm, string(subscription.PlanType))

	// Try cache first
	if cachedData, err := h.cacheService.Get(ctx, cacheKey); err == nil {
		var response SuggestionResponse
		if err := json.Unmarshal([]byte(cachedData), &response); err == nil {
			w.Header().Set("X-Cache", "HIT")
			respondWithJSON(w, http.StatusOK, response)
			return
		}
	}

	// Fetch all types of suggestions concurrently
	response := SuggestionResponse{}
	limit := 5 // Reduced limit for combined results

	// Using channels for concurrent fetching
	namesCh := make(chan []string)
	countriesCh := make(chan []string)
	categoriesCh := make(chan []string)
	citiesCh := make(chan []string)
	errorCh := make(chan error)

	go func() {
		var names []string
		err := h.db.Model(&models.Landmark{}).
			Select("DISTINCT name").
			Where("name ILIKE ?", "%"+searchTerm+"%").
			Limit(limit).
			Pluck("name", &names).Error
		if err != nil {
			errorCh <- err
			return
		}
		namesCh <- names
	}()

	go func() {
		var countries []string
		err := h.db.Model(&models.Landmark{}).
			Select("DISTINCT country").
			Where("country ILIKE ?", "%"+searchTerm+"%").
			Limit(limit).
			Pluck("country", &countries).Error
		if err != nil {
			errorCh <- err
			return
		}
		countriesCh <- countries
	}()

	go func() {
		var categories []string
		err := h.db.Model(&models.Landmark{}).
			Select("DISTINCT category").
			Where("category ILIKE ?", "%"+searchTerm+"%").
			Limit(limit).
			Pluck("category", &categories).Error
		if err != nil {
			errorCh <- err
			return
		}
		categoriesCh <- categories
	}()

	go func() {
		var cities []string
		err := h.db.Model(&models.Landmark{}).
			Select("DISTINCT city").
			Where("city ILIKE ?", "%"+searchTerm+"%").
			Limit(limit).
			Pluck("city", &cities).Error
		if err != nil {
			errorCh <- err
			return
		}
		citiesCh <- cities
	}()

	// Collect results with timeout
	timeout := time.After(3 * time.Second)
	for i := 0; i < 4; i++ {
		select {
		case names := <-namesCh:
			response.Names = names
		case countries := <-countriesCh:
			response.Countries = countries
		case categories := <-categoriesCh:
			response.Categories = categories
		case cities := <-citiesCh:
			response.Cities = cities
		case err := <-errorCh:
			respondWithError(w, http.StatusInternalServerError, "Error fetching suggestions: "+err.Error())
			return
		case <-timeout:
			respondWithError(w, http.StatusGatewayTimeout, "Timeout fetching suggestions")
			return
		}
	}

	// Cache the combined response
	h.cacheService.Set(ctx, cacheKey, response, 5*time.Minute)
	w.Header().Set("X-Cache", "MISS")
	respondWithJSON(w, http.StatusOK, response)
}

func (h *SuggestionsHandler) getCacheKey(parts ...string) string {
	key := "suggestions"
	for _, part := range parts {
		key += ":" + part
	}
	return key
}
