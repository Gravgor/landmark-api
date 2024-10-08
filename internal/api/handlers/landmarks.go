package handlers

import (
	"encoding/json"
	"landmark-api/internal/models"
	"net/http"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type LandmarkHandler struct {
	DB *gorm.DB
}

func (h *LandmarkHandler) GetLandmarks(w http.ResponseWriter, r *http.Request) {
	var landmarks []models.Landmark
	result := h.DB.Find(&landmarks)
	if result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(landmarks)
}

func (h *LandmarkHandler) GetLandmark(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	var landmark models.Landmark
	result := h.DB.First(&landmark, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			http.Error(w, "Landmark not found", http.StatusNotFound)
		} else {
			http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(landmark)
}

func (h *LandmarkHandler) GetLandmarksByCountry(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	country := params["country"]

	var landmarks []models.Landmark
	result := h.DB.Where("country ILIKE ?", country).Find(&landmarks)
	if result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(landmarks)
}

func (h *LandmarkHandler) CreateLandmark(w http.ResponseWriter, r *http.Request) {
	var landmark models.Landmark
	if err := json.NewDecoder(r.Body).Decode(&landmark); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result := h.DB.Create(&landmark)
	if result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(landmark)
}

func (h *LandmarkHandler) UpdateLandmark(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	var landmark models.Landmark
	if err := json.NewDecoder(r.Body).Decode(&landmark); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var existingLandmark models.Landmark
	result := h.DB.First(&existingLandmark, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			http.Error(w, "Landmark not found", http.StatusNotFound)
		} else {
			http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		}
		return
	}

	result = h.DB.Model(&existingLandmark).Updates(landmark)
	if result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existingLandmark)
}

func (h *LandmarkHandler) DeleteLandmark(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	result := h.DB.Delete(&models.Landmark{}, id)
	if result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}
	if result.RowsAffected == 0 {
		http.Error(w, "Landmark not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
