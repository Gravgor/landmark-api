package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"landmark-api/internal/models"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// Constants for the handler
const (
	minSimilarityThreshold = 0.3
	defaultCacheDuration   = 5 * time.Minute
	defaultLimit           = 10
	searchTimeout          = 3 * time.Second
)

// SearchResult represents the basic search result
type SearchResult struct {
	Value      string  `json:"value"`
	Similarity float64 `json:"similarity"`
}

// SuggestionResponse holds the structured response for different search types
type SuggestionResponse struct {
	Results []string `json:"results"`
}

// SuggestionsHandler handles all suggestion-related requests
type SuggestionsHandler struct {
	db           *gorm.DB
	cacheService CacheService
	config       *SuggestionsConfig
}

// SuggestionsConfig contains configuration for the suggestions handler
type SuggestionsConfig struct {
	MaxResults         int
	CacheDuration      time.Duration
	MinSimilarity      float64
	EnabledSearchTypes []string
	Weights            SearchWeights
}

// SearchWeights contains weights for different search methods
type SearchWeights struct {
	ExactMatch  float64
	Trigram     float64
	Metaphone   float64
	Levenshtein float64
}

// CacheService interface defines the required caching methods
type CacheService interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, duration time.Duration) error
}

// NewSuggestionsHandler creates a new instance of SuggestionsHandler
func NewSuggestionsHandler(db *gorm.DB, cacheService CacheService, config *SuggestionsConfig) (*SuggestionsHandler, error) {

	handler := &SuggestionsHandler{
		db:           db,
		cacheService: cacheService,
		config:       config,
	}

	if err := handler.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize handler: %v", err)
	}

	return handler, nil
}

func (h *SuggestionsHandler) GetSuggestions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract parameters
	vars := mux.Vars(r)
	searchType := vars["type"]
	searchTerm := r.URL.Query().Get("search")

	// Validate search type
	if !isValidSearchType(searchType) {
		respondWithError(w, http.StatusBadRequest, "Invalid search type")
		return
	}

	// Handle empty search term
	if searchTerm == "" {
		respondWithJSON(w, http.StatusOK, SuggestionResponse{Results: []string{}})
		return
	}

	// Perform search
	results, err := h.searchLandmarks(ctx, searchType, searchTerm)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error performing search")
		return
	}

	respondWithJSON(w, http.StatusOK, SuggestionResponse{Results: results})
}

func (h *SuggestionsHandler) searchLandmarks(ctx context.Context, searchType, searchTerm string) ([]string, error) {
	var results []string

	// Build the search condition based on search type
	column := getColumnForSearchType(searchType)
	if column == "" {
		return nil, fmt.Errorf("invalid search type")
	}

	// Clean and prepare the search term
	searchTerm = strings.TrimSpace(searchTerm)
	if searchTerm == "" {
		return []string{}, nil
	}

	// Create a more lenient search pattern
	searchPattern := "%" + strings.ToLower(searchTerm) + "%"

	// Query with additional logging and error handling
	query := h.db.WithContext(ctx).
		Model(&models.Landmark{}).
		Where(fmt.Sprintf("LOWER(%s) LIKE ?", column), searchPattern).
		Where("deleted_at IS NULL"). // Add soft delete check if you're using it
		Order(column).
		Distinct(column)

	err := query.Pluck(column, &results).Error
	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	// If no results found, try a more lenient search
	if len(results) == 0 {
		// Try searching with each word separately
		words := strings.Fields(searchTerm)
		for _, word := range words {
			pattern := "%" + strings.ToLower(word) + "%"
			var tempResults []string

			err := h.db.WithContext(ctx).
				Model(&models.Landmark{}).
				Where(fmt.Sprintf("LOWER(%s) LIKE ?", column), pattern).
				Where("deleted_at IS NULL").
				Order(column).
				Distinct(column).
				Pluck(column, &tempResults).
				Error

			if err != nil {
				continue
			}

			results = append(results, tempResults...)
		}

		// Remove duplicates if any were found in the second pass
		if len(results) > 0 {
			seen := make(map[string]bool)
			unique := make([]string, 0, len(results))
			for _, result := range results {
				if !seen[result] {
					seen[result] = true
					unique = append(unique, result)
				}
			}
			results = unique
		}
	}

	// Limit results if configured
	if h.config != nil && h.config.MaxResults > 0 && len(results) > h.config.MaxResults {
		results = results[:h.config.MaxResults]
	}

	return results, nil
}

// Utility functions
func getColumnForSearchType(searchType string) string {
	switch searchType {
	case "name":
		return "name"
	case "country":
		return "country"
	case "category":
		return "category"
	case "city":
		return "city"
	default:
		return ""
	}
}

func isValidSearchType(searchType string) bool {
	validTypes := map[string]bool{
		"name":     true,
		"country":  true,
		"category": true,
		"city":     true,
	}
	return validTypes[searchType]
}

func (h *SuggestionsHandler) getEmptyResponse(searchType string) SuggestionResponse {
	var response SuggestionResponse
	response.Results = []string{}
	return response
}

func (h *SuggestionsHandler) buildCacheKey(parts ...string) string {
	return fmt.Sprintf("suggestions:%s", strings.Join(parts, ":"))
}

func (h *SuggestionsHandler) cacheResponse(ctx context.Context, key string, response SuggestionResponse) error {
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}
	return h.cacheService.Set(ctx, key, string(data), h.config.CacheDuration)
}

// Initialize function for setting up necessary database extensions and indexes
func (h *SuggestionsHandler) Initialize() error {
	// Enable PostgreSQL extensions
	extensions := []string{
		"CREATE EXTENSION IF NOT EXISTS pg_trgm;",
		"CREATE EXTENSION IF NOT EXISTS fuzzystrmatch;",
		"CREATE EXTENSION IF NOT EXISTS unaccent;",
	}

	for _, ext := range extensions {
		if err := h.db.Exec(ext).Error; err != nil {
			return fmt.Errorf("failed to create extension: %v", err)
		}
	}

	// Create indexes for each searchable column
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_landmarks_name_trgm ON landmarks USING gin (name gin_trgm_ops);",
		"CREATE INDEX IF NOT EXISTS idx_landmarks_country_trgm ON landmarks USING gin (country gin_trgm_ops);",
		"CREATE INDEX IF NOT EXISTS idx_landmarks_category_trgm ON landmarks USING gin (category gin_trgm_ops);",
		"CREATE INDEX IF NOT EXISTS idx_landmarks_city_trgm ON landmarks USING gin (city gin_trgm_ops);",
	}

	for _, idx := range indexes {
		if err := h.db.Exec(idx).Error; err != nil {
			return fmt.Errorf("failed to create index: %v", err)
		}
	}

	return nil
}
