package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gorm.io/gorm"

	"landmark-api/internal/models"
	"landmark-api/internal/services"
)

type LandmarkHandler struct {
	landmarkService services.LandmarkService
	db              *gorm.DB
}

type QueryParams struct {
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
	Fields    []string
	Filters   map[string]string
}

func NewLandmarkHandler(landmarkService services.LandmarkService, db *gorm.DB) *LandmarkHandler {
	return &LandmarkHandler{
		landmarkService: landmarkService,
		db:              db,
	}
}

// GetLandmark - Enhanced with caching and field selection
func (h *LandmarkHandler) GetLandmark(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid landmark ID")
		return
	}

	// Try to get from cache first
	/*cacheKey := fmt.Sprintf("landmark:%s", id)
	cachedData, err := h.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		fmt.Fprint(w, cachedData)
		return
	}*/

	landmark, subscription, err := h.getLandmarkAndSubscription(ctx, id, w)
	if err != nil {
		return
	}

	response := h.prepareResponse(ctx, landmark, subscription, parseQueryParams(r))

	// Cache the response
	//jsonResponse, _ := json.Marshal(response)
	//h.cache.Set(ctx, cacheKey, jsonResponse, 1*time.Hour)

	respondWithJSON(w, http.StatusOK, response)
}

// ListLandmarks - Enhanced with filtering, sorting, and field selection
func (h *LandmarkHandler) ListLandmarks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	queryParams := parseQueryParams(r)

	subscription, ok := services.SubscriptionFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusForbidden, "Subscription not found")
		return
	}

	// Build the database query
	query := h.db.Model(&models.Landmark{})
	query = applyFilters(query, queryParams.Filters)
	query = applySorting(query, queryParams.SortBy, queryParams.SortOrder)

	var landmarks []models.Landmark
	if err := query.Offset(queryParams.Offset).Limit(queryParams.Limit).Find(&landmarks).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error fetching landmarks")
		return
	}

	response := h.processLandmarkList(ctx, landmarks, subscription, queryParams)
	respondWithJSON(w, http.StatusOK, response)
}

// ListLandmarksByCountry - Enhanced with filtering, sorting, and field selection
func (h *LandmarkHandler) ListLandmarksByCountry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	country := vars["country"]
	queryParams := parseQueryParams(r)

	subscription, ok := services.SubscriptionFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusForbidden, "Subscription not found")
		return
	}

	query := h.db.Model(&models.Landmark{}).Where("country = ?", country)
	query = applyFilters(query, queryParams.Filters)
	query = applySorting(query, queryParams.SortBy, queryParams.SortOrder)

	var landmarks []models.Landmark
	if err := query.Offset(queryParams.Offset).Limit(queryParams.Limit).Find(&landmarks).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error fetching landmarks")
		return
	}

	response := h.processLandmarkList(ctx, landmarks, subscription, queryParams)
	respondWithJSON(w, http.StatusOK, response)
}

func (h *LandmarkHandler) ListLandmarkByCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	category := vars["category"]
	queryParams := parseQueryParams(r)

	subscription, ok := services.SubscriptionFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusForbidden, "Subscription not found")
		return
	}

	query := h.db.Model(&models.Landmark{}).Where("category = ?", category)
	query = applyFilters(query, queryParams.Filters)
	query = applySorting(query, queryParams.SortBy, queryParams.SortOrder)
	var landmarks []models.Landmark
	if err := query.Offset(queryParams.Offset).Limit(queryParams.Limit).Find(&landmarks).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error fetching landmarks")
		return
	}

	response := h.processLandmarkList(ctx, landmarks, subscription, queryParams)
	respondWithJSON(w, http.StatusOK, response)

}

func (h *LandmarkHandler) ListLandmarksByName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	name := vars["name"]
	queryParams := parseQueryParams(r)

	// Get subscription from context
	subscription, ok := services.SubscriptionFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusForbidden, "Subscription not found")
		return
	}

	// Build the base query
	query := h.db.Model(&models.Landmark{}).Where("name ILIKE ?", "%"+name+"%")

	// Apply additional filters and sorting
	query = applyFilters(query, queryParams.Filters)
	query = applySorting(query, queryParams.SortBy, queryParams.SortOrder)

	// Execute the query
	var landmarks []models.Landmark
	if err := query.Offset(queryParams.Offset).Limit(queryParams.Limit).Find(&landmarks).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error fetching landmarks")
		return
	}

	// If no landmarks found, return empty result instead of error
	if len(landmarks) == 0 {
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"data": []interface{}{},
			"meta": map[string]interface{}{
				"total":  0,
				"limit":  queryParams.Limit,
				"offset": queryParams.Offset,
			},
		})
		return
	}

	// Process the landmarks list based on subscription and query parameters
	response := h.processLandmarkList(ctx, landmarks, subscription, queryParams)
	respondWithJSON(w, http.StatusOK, response)
}

// Helper functions

func parseQueryParams(r *http.Request) QueryParams {
	query := r.URL.Query()
	limit, _ := strconv.Atoi(query.Get("limit"))
	offset, _ := strconv.Atoi(query.Get("offset"))

	if limit == 0 {
		limit = 10
	}

	fields := []string{}
	if fieldStr := query.Get("fields"); fieldStr != "" {
		fields = strings.Split(fieldStr, ",")
	}

	filters := make(map[string]string)
	for k, v := range query {
		if k != "limit" && k != "offset" && k != "sort" && k != "fields" {
			filters[k] = v[0]
		}
	}

	sortBy := query.Get("sort")
	sortOrder := "asc"
	if strings.HasPrefix(sortBy, "-") {
		sortBy = strings.TrimPrefix(sortBy, "-")
		sortOrder = "desc"
	}

	return QueryParams{
		Limit:     limit,
		Offset:    offset,
		SortBy:    sortBy,
		SortOrder: sortOrder,
		Fields:    fields,
		Filters:   filters,
	}
}

func applyFilters(query *gorm.DB, filters map[string]string) *gorm.DB {
	for field, value := range filters {
		query = query.Where(fmt.Sprintf("%s = ?", field), value)
	}
	return query
}

func applySorting(query *gorm.DB, sortBy, sortOrder string) *gorm.DB {
	if sortBy != "" {
		query = query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))
	}
	return query
}

func (h *LandmarkHandler) prepareResponse(ctx context.Context, landmark *models.Landmark, subscription *models.Subscription, params QueryParams) interface{} {
	var response interface{}

	switch subscription.PlanType {
	case models.FreePlan:
		response = h.filterBasicLandmarkInfo(landmark)
	case models.ProPlan, models.EnterprisePlan:
		landmarkDetails, err := h.landmarkService.GetLandmarkDetails(ctx, landmark.ID, subscription.PlanType)
		if err != nil {
			return h.filterBasicLandmarkInfo(landmark)
		}
		response = h.mergeLandmarkAndDetails(landmark, landmarkDetails)
	}

	if len(params.Fields) > 0 {
		return filterFields(response, params.Fields)
	}

	return response
}

func filterFields(data interface{}, fields []string) map[string]interface{} {
	result := make(map[string]interface{})
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return result
	}

	for _, field := range fields {
		if value, exists := dataMap[field]; exists {
			result[field] = value
		}
	}
	return result
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

// Existing helper methods remain largely unchanged but adapted for GORM
func (h *LandmarkHandler) getLandmarkAndSubscription(ctx context.Context, id uuid.UUID, w http.ResponseWriter) (*models.Landmark, *models.Subscription, error) {
	_, ok := services.UserFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return nil, nil, fmt.Errorf("unauthorized")
	}

	subscription, ok := services.SubscriptionFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusForbidden, "Subscription not found")
		return nil, nil, fmt.Errorf("subscription not found")
	}

	var landmark models.Landmark
	if err := h.db.First(&landmark, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			respondWithError(w, http.StatusNotFound, "Landmark not found")
		} else {
			respondWithError(w, http.StatusInternalServerError, "Error fetching landmark")
		}
		return nil, nil, err
	}

	return &landmark, subscription, nil
}

func (h *LandmarkHandler) filterBasicLandmarkInfo(landmark *models.Landmark) map[string]interface{} {
	return map[string]interface{}{
		"id":          landmark.ID,
		"name":        landmark.Name,
		"description": landmark.Description,
		"country":     landmark.Country,
		"city":        landmark.City,
		"category":    landmark.Category,
		"latitude":    landmark.Latitude,
		"longitude":   landmark.Longitude,
		"image_url":   landmark.ImageUrl,
	}
}

// mergeLandmarkAndDetails combines landmark data with its details based on subscription
func (h *LandmarkHandler) mergeLandmarkAndDetails(landmark *models.Landmark, details *models.LandmarkDetail) map[string]interface{} {
	merged := h.filterBasicLandmarkInfo(landmark)
	weatherData, err := services.FetchWeatherData(landmark.Latitude, landmark.Longitude)
	if err != nil {
		fmt.Print("Error with weather")
	}
	if details != nil {
		additionalInfo := map[string]interface{}{
			"opening_hours":           details.OpeningHours,
			"ticket_prices":           details.TicketPrices,
			"historical_significance": details.HistoricalSignificance,
			"visitor_tips":            details.VisitorTips,
			"accessibility_info":      details.AccessibilityInfo,
			"weather_info":            weatherData,
		}

		// Add weather info for enterprise plan

		for k, v := range additionalInfo {
			merged[k] = v
		}
	}

	return merged
}

// processLandmarkList handles the processing of multiple landmarks based on subscription and query parameters
func (h *LandmarkHandler) processLandmarkList(ctx context.Context, landmarks []models.Landmark, subscription *models.Subscription, params QueryParams) map[string]interface{} {
	var processedLandmarks []map[string]interface{}

	for _, landmark := range landmarks {
		var landmarkData map[string]interface{}

		switch subscription.PlanType {
		case models.FreePlan:
			landmarkData = h.filterBasicLandmarkInfo(&landmark)
		case models.ProPlan, models.EnterprisePlan:
			details, err := h.landmarkService.GetLandmarkDetails(ctx, landmark.ID, subscription.PlanType)
			if err != nil {
				landmarkData = h.filterBasicLandmarkInfo(&landmark)
			} else {
				landmarkData = h.mergeLandmarkAndDetails(&landmark, details)
			}
		}

		// Apply field filtering if specified
		if len(params.Fields) > 0 {
			landmarkData = filterFields(landmarkData, params.Fields)
		}

		processedLandmarks = append(processedLandmarks, landmarkData)
	}

	// Get total count for pagination
	var totalCount int64
	h.db.Model(&models.Landmark{}).Count(&totalCount)

	return map[string]interface{}{
		"data": processedLandmarks,
		"meta": map[string]interface{}{
			"total":  totalCount,
			"limit":  params.Limit,
			"offset": params.Offset,
		},
	}
}
