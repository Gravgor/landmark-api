package handlers

import (
	"landmark-api/internal/services"
	"log"
	"net/http"
)

type LandmarkStatsHandler struct {
	landmarkStatsService services.LandmarkStatsService
}

func NewLandmarkStatsHandler(landmarkStatsService services.LandmarkStatsService) *LandmarkStatsHandler {
	return &LandmarkStatsHandler{
		landmarkStatsService: landmarkStatsService,
	}
}

func (h *LandmarkStatsHandler) GetLandmarkStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := h.landmarkStatsService.GetLandmarkStats(ctx)
	if err != nil {
		log.Printf("Error fetching landmark stats: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Error fetching landmark stats")
		return
	}

	respondWithJSON(w, http.StatusOK, stats)
}
