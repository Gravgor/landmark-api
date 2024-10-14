package handlers

import (
	"encoding/json"
	"landmark-api/internal/services"
	"net/http"
	"time"
)

type RequestLogHandler struct {
	logService services.RequestLogService
}

func NewRequestLogHandler(logService services.RequestLogService) *RequestLogHandler {
	return &RequestLogHandler{
		logService: logService,
	}
}

func (h *RequestLogHandler) GetUserLogs(w http.ResponseWriter, r *http.Request) {
	user, ok := services.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse time range from query parameters
	from, to := getTimeRange(r)

	logs, err := h.logService.GetUserLogs(user.ID.String(), from, to)
	if err != nil {
		http.Error(w, "Error fetching logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func getTimeRange(r *http.Request) (time.Time, time.Time) {
	now := time.Now()
	from := now.AddDate(0, -1, 0) // Default to last 30 days
	to := now

	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if parsedFrom, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = parsedFrom
		}
	}

	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if parsedTo, err := time.Parse(time.RFC3339, toStr); err == nil {
			to = parsedTo
		}
	}

	return from, to
}
