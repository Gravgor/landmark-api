package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"gorm.io/gorm"
)

type HealthCheckResponse struct {
	Status           string            `json:"status"`
	Database         string            `json:"database"`
	ExternalServices map[string]string `json:"external_services"`
}

// HealthCheckHandler checks API health, database connection, and external services
func HealthCheckHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := HealthCheckResponse{
			ExternalServices: make(map[string]string),
		}

		// Check database connection
		sqlDB, err := db.DB()
		if err != nil {
			response.Status = "API is running"
			response.Database = "Database connection failed"
			respondWithJSON(w, http.StatusInternalServerError, response)
			return
		}

		if err := sqlDB.Ping(); err != nil {
			response.Status = "API is running"
			response.Database = "Database connection failed"
			respondWithJSON(w, http.StatusInternalServerError, response)
			return
		}

		response.Status = "API is running"
		response.Database = "Database connection is healthy"

		// Check external service (example: Weather API)
		weatherAPIURL := "http://api.openweathermap.org/data/2.5/weather?q=London&appid=d0e23c5d2a622321138d993e9e7f9f23"
		externalServiceStatus := checkExternalService(weatherAPIURL)
		response.ExternalServices["Weather API"] = externalServiceStatus

		// Respond with API, database, and external services status
		respondWithJSON(w, http.StatusOK, response)
	}
}

// respondWithJSON sends a JSON response
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

// checkExternalService checks the status of an external service
func checkExternalService(url string) string {
	client := http.Client{
		Timeout: 5 * time.Second, // Set a timeout for the request
	}

	resp, err := client.Get(url)
	if err != nil {
		return "Unreachable"
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return "Available"
	}
	return "Unavailable"
}
