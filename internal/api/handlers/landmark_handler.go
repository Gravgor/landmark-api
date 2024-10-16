package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"gorm.io/gorm"

	"landmark-api/internal/models"
	"landmark-api/internal/services"
)

type LandmarkHandler struct {
	landmarkService services.LandmarkService
	auditService    services.AuditLogService
	cacheService    services.CacheService
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

func NewLandmarkHandler(landmarkService services.LandmarkService, as services.AuditLogService, cs services.CacheService, db *gorm.DB) *LandmarkHandler {
	return &LandmarkHandler{
		landmarkService: landmarkService,
		cacheService:    cs,
		auditService:    as,
		db:              db,
	}
}

func (h *LandmarkHandler) getCacheKey(params ...string) string {
	return fmt.Sprintf("landmark:%s", strings.Join(params, ":"))
}

// GetLandmark godoc
// @Summary Get a landmark by ID
// @Description Get detailed information about a landmark
// @Tags landmarks
// @Accept json
// @Produce json
// @Param id path string true "Landmark ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/landmarks/{id} [get]
func (h *LandmarkHandler) GetLandmark(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid landmark ID")
		return
	}

	subscription, ok := services.SubscriptionFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusForbidden, "Subscription not found")
		return
	}

	// Try to get from cache
	cacheKey := h.getCacheKey("id", idStr, string(subscription.PlanType))
	if cachedData, err := h.cacheService.Get(ctx, cacheKey); err == nil {
		var response interface{}
		if err := json.Unmarshal([]byte(cachedData), &response); err == nil {
			w.Header().Set("X-Cache", "HIT")
			respondWithJSON(w, http.StatusOK, response)
			return
		}
	}

	landmark, subscription, err := h.getLandmarkAndSubscription(ctx, id, w)
	if err != nil {
		return
	}

	response := h.prepareResponse(ctx, landmark, subscription, parseQueryParams(r))

	// Cache the response
	h.cacheService.Set(ctx, cacheKey, response, 15*time.Minute)
	w.Header().Set("X-Cache", "MISS")
	respondWithJSON(w, http.StatusOK, response)
}

// ListLandmarks godoc
// @Summary List landmarks
// @Description Get a list of landmarks with optional filtering and sorting
// @Tags landmarks
// @Accept json
// @Produce json
// @Param limit query int false "Number of items to return"
// @Param offset query int false "Number of items to skip"
// @Param sort query string false "Sort field and order (e.g., '-name' for descending)"
// @Param fields query string false "Comma-separated list of fields to include"
// @Success 200 {object} map[string]interface{}
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/landmarks [get]
func (h *LandmarkHandler) ListLandmarks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	queryParams := parseQueryParams(r)

	subscription, ok := services.SubscriptionFromContext(ctx)
	if !ok {
		respondWithError(w, http.StatusForbidden, "Subscription not found")
		return
	}

	// Generate cache key based on query parameters
	cacheKey := h.getCacheKey("list",
		fmt.Sprintf("limit:%d", queryParams.Limit),
		fmt.Sprintf("offset:%d", queryParams.Offset),
		fmt.Sprintf("sort:%s:%s", queryParams.SortBy, queryParams.SortOrder),
		string(subscription.PlanType))

	// Try to get from cache
	if cachedData, err := h.cacheService.Get(ctx, cacheKey); err == nil {
		var response interface{}
		if err := json.Unmarshal([]byte(cachedData), &response); err == nil {
			w.Header().Set("X-Cache", "HIT")
			respondWithJSON(w, http.StatusOK, response)
			return
		}
	}

	query := h.db.Model(&models.Landmark{}).Preload("Images")
	query = applyFilters(query, queryParams.Filters)
	query = applySorting(query, queryParams.SortBy, queryParams.SortOrder)

	var landmarks []models.Landmark
	if err := query.Offset(queryParams.Offset).Limit(queryParams.Limit).Find(&landmarks).Error; err != nil {
		fmt.Print(err)
		respondWithError(w, http.StatusInternalServerError, "Error fetching landmarks")
		return
	}

	response := h.processLandmarkList(ctx, landmarks, subscription, queryParams)

	// Cache the response
	h.cacheService.Set(ctx, cacheKey, response, 15*time.Minute)
	w.Header().Set("X-Cache", "MISS")
	respondWithJSON(w, http.StatusOK, response)
}

func (h *LandmarkHandler) ListAdminLandmarks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := strconv.Atoi(r.URL.Query().Get("per_page"))
	if err != nil || perPage < 1 {
		perPage = 10 // Default to 10 items per page
	}

	searchTerm := r.URL.Query().Get("search")
	category := r.URL.Query().Get("category")

	// Fetch landmarks with pagination, search, and category filter
	landmarks, total, err := h.landmarkService.GetLandmarksWithFilters(ctx, page, perPage, searchTerm, category)
	if err != nil {
		log.Printf("Error fetching landmarks: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Error fetching landmarks")
		return
	}

	// Prepare the response with full landmark information
	var fullLandmarks []map[string]interface{}
	for _, landmark := range landmarks {
		// Fetch admin details for each landmark
		details, err := h.landmarkService.GetLandmarkAdminDetails(ctx, landmark.ID)
		if err != nil {
			log.Printf("Error fetching details for landmark %s: %v", landmark.ID, err)
			// Decide whether to skip this landmark or continue with partial data
			continue
		}

		fullLandmark := map[string]interface{}{
			"id":          landmark.ID,
			"name":        landmark.Name,
			"description": landmark.Description,
			"latitude":    landmark.Latitude,
			"longitude":   landmark.Longitude,
			"country":     landmark.Country,
			"city":        landmark.City,
			"category":    landmark.Category,
			"image_url":   landmark.ImageUrl,
			"images":      landmark.Images,
			"created_at":  landmark.CreatedAt,
			"updated_at":  landmark.UpdatedAt,
		}

		// Add admin details
		if details != nil {
			fullLandmark["opening_hours"] = details.OpeningHours
			fullLandmark["ticket_prices"] = details.TicketPrices
			fullLandmark["historical_significance"] = details.HistoricalSignificance
			fullLandmark["visitor_tips"] = details.VisitorTips
			fullLandmark["accessibility_info"] = details.AccessibilityInfo
		}

		fullLandmarks = append(fullLandmarks, fullLandmark)
	}

	response := map[string]interface{}{
		"landmarks": fullLandmarks,
		"total":     total,
		"page":      page,
		"per_page":  perPage,
	}

	respondWithJSON(w, http.StatusOK, response)
}

// ListLandmarksByCountry godoc
// @Summary List landmarks by country
// @Description Get a list of landmarks for a specific country
// @Tags landmarks
// @Accept json
// @Produce json
// @Param country path string true "Country name"
// @Param limit query int false "Number of items to return"
// @Param offset query int false "Number of items to skip"
// @Param sort query string false "Sort field and order (e.g., '-name' for descending)"
// @Param fields query string false "Comma-separated list of fields to include"
// @Success 200 {object} map[string]interface{}
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/landmarks/country/{country} [get]
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

	// Generate cache key
	cacheKey := h.getCacheKey("country", country,
		fmt.Sprintf("limit:%d", queryParams.Limit),
		fmt.Sprintf("offset:%d", queryParams.Offset),
		fmt.Sprintf("sort:%s:%s", queryParams.SortBy, queryParams.SortOrder),
		string(subscription.PlanType))

	// Try to get from cache
	if cachedData, err := h.cacheService.Get(ctx, cacheKey); err == nil {
		var response interface{}
		if err := json.Unmarshal([]byte(cachedData), &response); err == nil {
			w.Header().Set("X-Cache", "HIT")
			respondWithJSON(w, http.StatusOK, response)
			return
		}
	}

	query := h.db.Model(&models.Landmark{}).Where("country = ?", country).Preload("Images")
	query = applyFilters(query, queryParams.Filters)
	query = applySorting(query, queryParams.SortBy, queryParams.SortOrder)

	var landmarks []models.Landmark
	if err := query.Offset(queryParams.Offset).Limit(queryParams.Limit).Find(&landmarks).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error fetching landmarks")
		return
	}

	response := h.processLandmarkList(ctx, landmarks, subscription, queryParams)

	// Cache the response
	h.cacheService.Set(ctx, cacheKey, response, 15*time.Minute)
	w.Header().Set("X-Cache", "MISS")
	respondWithJSON(w, http.StatusOK, response)
}

// ListLandmarkByCategory godoc
// @Summary List landmarks by category
// @Description Get a list of landmarks for a specific category
// @Tags landmarks
// @Accept json
// @Produce json
// @Param category path string true "Category name"
// @Param limit query int false "Number of items to return"
// @Param offset query int false "Number of items to skip"
// @Param sort query string false "Sort field and order (e.g., '-name' for descending)"
// @Param fields query string false "Comma-separated list of fields to include"
// @Success 200 {object} map[string]interface{}
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/landmarks/category/{category} [get]
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

	// Generate cache key based on category, query parameters, and subscription type
	cacheKey := h.getCacheKey("category", category,
		fmt.Sprintf("limit:%d", queryParams.Limit),
		fmt.Sprintf("offset:%d", queryParams.Offset),
		fmt.Sprintf("sort:%s:%s", queryParams.SortBy, queryParams.SortOrder),
		string(subscription.PlanType))

	// Try to get from cache first
	if cachedData, err := h.cacheService.Get(ctx, cacheKey); err == nil {
		var response interface{}
		if err := json.Unmarshal([]byte(cachedData), &response); err == nil {
			w.Header().Set("X-Cache", "HIT")
			respondWithJSON(w, http.StatusOK, response)
			return
		}
		// If unmarshal fails, log the error but continue to fetch from database
		log.Printf("Error unmarshaling cached data: %v", err)
	}

	// Cache miss or error - fetch from database
	query := h.db.Model(&models.Landmark{}).Where("category = ?", category).Preload("Images")
	query = applyFilters(query, queryParams.Filters)
	query = applySorting(query, queryParams.SortBy, queryParams.SortOrder)

	var landmarks []models.Landmark
	if err := query.Offset(queryParams.Offset).Limit(queryParams.Limit).Find(&landmarks).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error fetching landmarks")
		return
	}

	// If no landmarks found, return empty result instead of error
	if len(landmarks) == 0 {
		emptyResponse := map[string]interface{}{
			"data": []interface{}{},
			"meta": map[string]interface{}{
				"total":  0,
				"limit":  queryParams.Limit,
				"offset": queryParams.Offset,
			},
		}

		// Cache the empty response too
		h.cacheService.Set(ctx, cacheKey, emptyResponse, 15*time.Minute)
		w.Header().Set("X-Cache", "MISS")
		respondWithJSON(w, http.StatusOK, emptyResponse)
		return
	}

	// Process the landmarks list based on subscription and query parameters
	response := h.processLandmarkList(ctx, landmarks, subscription, queryParams)

	// Cache the successful response
	if err := h.cacheService.Set(ctx, cacheKey, response, 15*time.Minute); err != nil {
		// Log cache set error but continue with response
		log.Printf("Error setting cache: %v", err)
	}

	w.Header().Set("X-Cache", "MISS")
	respondWithJSON(w, http.StatusOK, response)
}

// Define a struct for the search request
type SearchRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Radius    float64 `json:"radius"` // in kilometers
}

// Function to calculate distance using Haversine formula
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Radius of Earth in kilometers
	dLat := (lat2 - lat1) * (math.Pi / 180)
	dLon := (lon2 - lon1) * (math.Pi / 180)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*(math.Pi/180))*math.Cos(lat2*(math.Pi/180))*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// SearchLandmarks godoc
// @Summary Search landmarks by proximity
// @Description Search for landmarks within a given radius of a point
// @Tags landmarks
// @Accept json
// @Produce json
// @Param request body SearchRequest true "Search parameters"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/landmarks/search [post]
func (h *LandmarkHandler) SearchLandmarks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	subscription, ok := services.SubscriptionFromContext(ctx)
	if !ok || subscription.PlanType != models.ProPlan {
		respondWithError(w, http.StatusForbidden, "Forbidden: Pro subscription required")
		return
	}
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	var landmarks []models.Landmark
	if err := h.db.Find(&landmarks).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error fetching landmarks")
		return
	}

	var results []models.Landmark
	for _, landmark := range landmarks {
		distance := haversine(req.Latitude, req.Longitude, landmark.Latitude, landmark.Longitude)
		if distance <= req.Radius {
			results = append(results, landmark)
		}
	}

	response := h.processLandmarkList(ctx, results, subscription, QueryParams{
		Limit:     len(results),        // Set limit to the number of results found
		Offset:    0,                   // No offset for this search
		SortBy:    "",                  // No specific sorting needed
		SortOrder: "asc",               // Default order
		Fields:    []string{},          // No field filtering specified
		Filters:   map[string]string{}, // No filters
	})

	respondWithJSON(w, http.StatusOK, response)
}

// ListLandmarksByName godoc
// @Summary List landmarks by name
// @Description Get a list of landmarks matching a given name (partial match)
// @Tags landmarks
// @Accept json
// @Produce json
// @Param name path string true "Landmark name (partial)"
// @Param limit query int false "Number of items to return"
// @Param offset query int false "Number of items to skip"
// @Param sort query string false "Sort field and order (e.g., '-name' for descending)"
// @Param fields query string false "Comma-separated list of fields to include"
// @Success 200 {object} map[string]interface{}
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/landmarks/name/{name} [get]
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

	cacheKey := h.getCacheKey("name", name,
		fmt.Sprintf("limit:%d", queryParams.Limit),
		fmt.Sprintf("offset:%d", queryParams.Offset),
		fmt.Sprintf("sort:%s:%s", queryParams.SortBy, queryParams.SortOrder),
		string(subscription.PlanType))

	if cachedData, err := h.cacheService.Get(ctx, cacheKey); err == nil {
		var response interface{}
		if err := json.Unmarshal([]byte(cachedData), &response); err == nil {
			w.Header().Set("X-Cache", "HIT")
			respondWithJSON(w, http.StatusOK, response)
			return
		}
	}

	// Build the base query
	query := h.db.Model(&models.Landmark{}).Where("name ILIKE ?", "%"+name+"%").Preload("Images")

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
	h.cacheService.Set(ctx, cacheKey, response, 15*time.Minute)
	w.Header().Set("X-Cache", "MISS")
	respondWithJSON(w, http.StatusOK, response)
}

func (h *LandmarkHandler) CreateLandmark(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	var landmarkData struct {
		Landmark       models.Landmark       `json:"landmark"`
		LandmarkDetail models.LandmarkDetail `json:"landmark_detail"`
		ImageURLs      []string              `json:"image_urls"`
	}

	if err := json.NewDecoder(r.Body).Decode(&landmarkData); err != nil {
		log.Printf("Error decoding JSON: %v", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Start a database transaction
	tx := h.db.Begin()
	if tx.Error != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to start database transaction")
		return
	}

	// Create the Landmark
	landmarkData.Landmark.ID = uuid.New() // Generate a new UUID for the landmark

	if err := tx.Create(&landmarkData.Landmark).Error; err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "Failed to create landmark")
		return
	}

	// Create LandmarkImage entries
	for _, url := range landmarkData.ImageURLs {
		landmarkImage := models.LandmarkImage{
			ID:         uuid.New(),
			LandmarkID: landmarkData.Landmark.ID,
			ImageURL:   url,
		}
		if err := tx.Create(&landmarkImage).Error; err != nil {
			tx.Rollback()
			respondWithError(w, http.StatusInternalServerError, "Failed to create landmark image")
			return
		}
	}

	// Create the LandmarkDetail
	landmarkData.LandmarkDetail.ID = uuid.New() // Generate a new UUID for the landmark detail
	landmarkData.LandmarkDetail.LandmarkID = landmarkData.Landmark.ID
	if err := tx.Create(&landmarkData.LandmarkDetail).Error; err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "Failed to create landmark details")
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	// Fetch the created landmark with its images
	var createdLandmark models.Landmark
	if err := h.db.Preload("Images").First(&createdLandmark, landmarkData.Landmark.ID).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch created landmark")
		return
	}

	adminID := 0
	err := h.auditService.CreateAuditLog(r.Context(), adminID, "CREATE", "LANDMARK", createdLandmark.ID.String(), "Created landmark")
	if err != nil {
		log.Printf("Failed to create audit log: %v", err)
	}

	// Prepare the response
	response := h.mergeLandmarkAndDetails(&createdLandmark, &landmarkData.LandmarkDetail)

	respondWithJSON(w, http.StatusCreated, response)
}

func (h *LandmarkHandler) AdminEditHandler(w http.ResponseWriter, r *http.Request) {
	// Extract landmark ID from the URL
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid landmark ID")
		return
	}

	// Decode the request body
	var updateData struct {
		Landmark struct {
			Name        string  `json:"name"`
			Description string  `json:"description"`
			Latitude    float64 `json:"latitude"`
			Longitude   float64 `json:"longitude"`
			Country     string  `json:"country"`
			City        string  `json:"city"`
			Category    string  `json:"category"`
		} `json:"landmark"`
		LandmarkDetail struct {
			OpeningHours           string `json:"opening_hours"`
			TicketPrices           string `json:"ticket_prices"`
			HistoricalSignificance string `json:"historical_significance"`
			VisitorTips            string `json:"visitor_tips"`
			AccessibilityInfo      string `json:"accessibility_info"`
		} `json:"landmark_detail"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Start a database transaction
	tx := h.db.Begin()
	if tx.Error != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to start database transaction")
		return
	}

	// Update the Landmark
	if err := tx.Model(&models.Landmark{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name":        updateData.Landmark.Name,
		"description": updateData.Landmark.Description,
		"latitude":    updateData.Landmark.Latitude,
		"longitude":   updateData.Landmark.Longitude,
		"country":     updateData.Landmark.Country,
		"city":        updateData.Landmark.City,
		"category":    updateData.Landmark.Category,
	}).Error; err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "Failed to update landmark")
		return
	}

	// Update the LandmarkDetail
	if err := tx.Model(&models.LandmarkDetail{}).Where("landmark_id = ?", id).Updates(map[string]interface{}{
		"opening_hours":           updateData.LandmarkDetail.OpeningHours,
		"ticket_prices":           updateData.LandmarkDetail.TicketPrices,
		"historical_significance": updateData.LandmarkDetail.HistoricalSignificance,
		"visitor_tips":            updateData.LandmarkDetail.VisitorTips,
		"accessibility_info":      updateData.LandmarkDetail.AccessibilityInfo,
	}).Error; err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "Failed to update landmark details")
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	// Fetch the updated landmark with its details
	var updatedLandmark models.Landmark
	var updatedDetails models.LandmarkDetail

	if err := h.db.Preload("Images").First(&updatedLandmark, id).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch updated landmark")
		return
	}

	if err := h.db.First(&updatedDetails, "landmark_id = ?", id).Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch updated landmark details")
		return
	}

	// Prepare the response
	response := h.mergeLandmarkAndDetails(&updatedLandmark, &updatedDetails)

	respondWithJSON(w, http.StatusOK, response)
}

func (h *LandmarkHandler) AdminDeleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid landmark ID")
		return
	}

	tx := h.db.Begin()
	if tx.Error != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to start database transaction")
		return
	}

	if err := tx.Where("landmark_id = ?", id).Delete(&models.LandmarkImage{}).Error; err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "Failed to delete associated images")
		return
	}

	if err := tx.Where("landmark_id = ?", id).Delete(&models.LandmarkDetail{}).Error; err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "Failed to delete landmark details")
		return
	}

	if err := tx.Delete(&models.Landmark{}, id).Error; err != nil {
		tx.Rollback()
		respondWithError(w, http.StatusInternalServerError, "Failed to delete landmark")
		return
	}

	if err := tx.Commit().Error; err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	cacheKey := h.getCacheKey("id", id.String())
	if err := h.cacheService.Delete(r.Context(), cacheKey); err != nil {
		log.Printf("Failed to delete cache entry: %v", err)
	}

	// Respond with a success message
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Landmark deleted successfully"})
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
	allowedSortBy := map[string]bool{
		"name":    true,
		"city":    true,
		"country": true,
	}

	allowedSortOrder := map[string]bool{
		"asc":  true,
		"desc": true,
	}

	if allowedSortBy[sortBy] && allowedSortOrder[sortOrder] {
		query = query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))
	} else {
		// Handle invalid sortBy or sortOrder, e.g., set default or return an error
		query = query.Order("name asc") // Default sorting
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
	if err := h.db.Preload("Images").First(&landmark, "id = ?", id).Error; err != nil {
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
		"images":      landmark.Images,
	}
}

// mergeLandmarkAndDetails combines landmark data with its details based on subscription
func (h *LandmarkHandler) mergeLandmarkAndDetails(landmark *models.Landmark, details *models.LandmarkDetail) map[string]interface{} {
	merged := h.filterBasicLandmarkInfo(landmark)

	// Add image information
	merged["images"] = landmark.Images

	// Fetch weather data
	weatherData, err := services.FetchWeatherData(landmark.Latitude, landmark.Longitude)
	if err != nil {
		log.Printf("Error fetching weather data: %v", err)
		weatherData = nil
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

		// Add additional info based on subscription level
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
