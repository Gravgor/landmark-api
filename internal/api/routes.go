package api

import (
	"landmark-api/internal/api/handlers"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func SetupRoutes(db *gorm.DB) *mux.Router {
	router := mux.NewRouter()

	landmarkHandler := &handlers.LandmarkHandler{DB: db}

	router.HandleFunc("/api/landmarks", landmarkHandler.GetLandmarks).Methods("GET")
	router.HandleFunc("/api/landmarks/{id}", landmarkHandler.GetLandmark).Methods("GET")
	router.HandleFunc("/api/landmarks/country/{country}", landmarkHandler.GetLandmarksByCountry).Methods("GET")
	router.HandleFunc("/api/landmarks", landmarkHandler.CreateLandmark).Methods("POST")
	router.HandleFunc("/api/landmarks/{id}", landmarkHandler.UpdateLandmark).Methods("PUT")
	router.HandleFunc("/api/landmarks/{id}", landmarkHandler.DeleteLandmark).Methods("DELETE")

	return router
}
