package handlers

import (
	"encoding/json"
	"landmark-api/internal/models"
	"landmark-api/internal/services"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type LandmarkHandler struct {
	landmarkService services.LandmarkService
}

func NewLandmarkHandler(landmarkService services.LandmarkService) *LandmarkHandler {
	return &LandmarkHandler{landmarkService: landmarkService}
}

// GetLandmark - Fetch basic or detailed landmark info based on user's subscription tier
func (h *LandmarkHandler) GetLandmark(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid landmark ID", http.StatusBadRequest)
		return
	}

	_, ok := services.UserFromContext(ctx)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	subscription, ok := services.SubscriptionFromContext(ctx)
	if !ok {
		http.Error(w, "Subscription not found", http.StatusForbidden)
		return
	}

	landmark, err := h.landmarkService.GetLandmark(ctx, id)
	if err != nil {
		http.Error(w, "Error fetching landmark", http.StatusInternalServerError)
		return
	}

	if landmark == nil {
		http.Error(w, "Landmark not found", http.StatusNotFound)
		return
	}

	if subscription.PlanType == models.FreePlan {
		basicInfo := h.filterBasicLandmarkInfo(landmark)
		json.NewEncoder(w).Encode(basicInfo)
		return
	}

	if subscription.PlanType == models.ProPlan || subscription.PlanType == models.EnterprisePlan {
		landmarkDetails, err := h.landmarkService.GetLandmarkDetails(ctx, id, subscription.PlanType)
		if err != nil {
			http.Error(w, "Error fetching landmark details", http.StatusInternalServerError)
			return
		}

		detailedInfo := h.mergeLandmarkAndDetails(landmark, landmarkDetails)
		json.NewEncoder(w).Encode(detailedInfo)
		return
	}
}

// ListLandmarks - Fetch landmarks list with limited information for Free tier
func (h *LandmarkHandler) ListLandmarks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit, offset := ParsePaginationParams(r)

	landmarks, err := h.landmarkService.ListLandmarks(ctx, limit, offset)
	if err != nil {
		http.Error(w, "Error fetching landmarks", http.StatusInternalServerError)
		return
	}

	subscription, ok := services.SubscriptionFromContext(ctx)
	if !ok {
		http.Error(w, "Subscription not found", http.StatusForbidden)
		return
	}

	if subscription.PlanType == models.FreePlan {
		for i := range landmarks {
			landmarks[i] = h.filterBasicLandmarkInfo(&landmarks[i])
		}
	}

	json.NewEncoder(w).Encode(landmarks)
}

// GetLandmarkDetails - Only accessible by Pro and Enterprise tier
func (h *LandmarkHandler) GetLandmarkDetails(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid landmark ID", http.StatusBadRequest)
		return
	}

	subscription, ok := services.SubscriptionFromContext(ctx)
	if !ok || subscription.PlanType == models.FreePlan {
		http.Error(w, "Upgrade your subscription to access detailed info", http.StatusForbidden)
		return
	}

	landmarkDetails, err := h.landmarkService.GetLandmarkDetails(ctx, id, subscription.PlanType)
	if err != nil {
		http.Error(w, "Error fetching landmark details", http.StatusInternalServerError)
		return
	}

	if landmarkDetails == nil {
		http.Error(w, "Landmark details not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(landmarkDetails)
}

// filterBasicLandmarkInfo - Utility function to filter basic landmark info for Free tier
func (h *LandmarkHandler) filterBasicLandmarkInfo(landmark *models.Landmark) models.Landmark {
	return models.Landmark{
		ID:          landmark.ID,
		Name:        landmark.Name,
		Description: landmark.Description, // Limit description if needed
		Country:     landmark.Country,
		City:        landmark.City,
		Category:    landmark.Category,
		Latitude:    landmark.Latitude,
		Longitude:   landmark.Longitude,
		// Basic info only
	}
}

// mergeLandmarkAndDetails - Utility to merge basic and detailed information for Pro/Enterprise users
func (h *LandmarkHandler) mergeLandmarkAndDetails(landmark *models.Landmark, details *models.LandmarkDetail) map[string]interface{} {
	return map[string]interface{}{
		"id":                      landmark.ID,
		"name":                    landmark.Name,
		"description":             landmark.Description,
		"country":                 landmark.Country,
		"city":                    landmark.City,
		"category":                landmark.Category,
		"latitude":                landmark.Latitude,
		"longitude":               landmark.Longitude,
		"opening_hours":           details.OpeningHours,
		"ticket_prices":           details.TicketPrices,
		"historical_significance": details.HistoricalSignificance,
		"visitor_tips":            details.VisitorTips,
		"accessibility_info":      details.AccessibilityInfo,
	}
}

// Utility to parse pagination params from query
func ParsePaginationParams(r *http.Request) (limit, offset int) {
	limit = 10
	offset = 0
	query := r.URL.Query()

	if limitParam := query.Get("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil {
			limit = parsedLimit
		}
	}

	if offsetParam := query.Get("offset"); offsetParam != "" {
		if parsedOffset, err := strconv.Atoi(offsetParam); err == nil {
			offset = parsedOffset
		}
	}

	return limit, offset
}
