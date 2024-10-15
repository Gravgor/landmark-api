package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"
	"time"
)

// UptimeHandler handles HTTP requests related to uptime
type UptimeHandler struct {
	service *UptimeService
}

// NewUptimeHandler creates a new UptimeHandler
func NewUptimeHandler(service *UptimeService) *UptimeHandler {
	return &UptimeHandler{service: service}
}

// ServeHTTP handles the HTTP request for uptime information
func (h *UptimeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uptimeData := h.service.GetUptimeData()
	anomalies := h.service.GetAnomalies()

	response := UptimeResponse{
		Uptime:           uptimeData.Uptime,
		UptimePercentage: fmt.Sprintf("%.2f%%", uptimeData.Uptime),
		Description:      getUptimeDescription(uptimeData.Uptime),
		Status:           getUptimeStatus(uptimeData.Uptime),
		TotalUptime:      uptimeData.TotalUptime.String(),
		LastDowntime:     uptimeData.LastDowntime.Format(time.RFC3339),
		Anomalies:        anomalies,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UptimeService manages uptime tracking logic and anomaly detection
type UptimeService struct {
	startTime       time.Time
	downtime        time.Duration
	lastDowntime    time.Time
	responseTimes   []time.Duration
	errorCount      int
	totalRequests   int
	anomalies       []Anomaly
	mu              sync.RWMutex
	anomalyDetector *AnomalyDetector
}

// NewUptimeService creates a new UptimeService
func NewUptimeService() *UptimeService {
	service := &UptimeService{
		startTime:       time.Now(),
		responseTimes:   make([]time.Duration, 0, 1000), // Keep last 1000 response times
		anomalyDetector: NewAnomalyDetector(),
	}
	go service.monitorAnomaly()
	return service
}

// RecordDowntime records a period of downtime
func (s *UptimeService) RecordDowntime(duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.downtime += duration
	s.lastDowntime = time.Now()
}

// RecordRequest records a request's response time and status
func (s *UptimeService) RecordRequest(responseTime time.Duration, isError bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responseTimes = append(s.responseTimes, responseTime)
	if len(s.responseTimes) > 1000 {
		s.responseTimes = s.responseTimes[1:]
	}
	s.totalRequests++
	if isError {
		s.errorCount++
	}
}

// GetUptimeData returns current uptime data
func (s *UptimeService) GetUptimeData() UptimeData {
	s.mu.RLock()
	defer s.mu.RUnlock()
	totalTime := time.Since(s.startTime)
	uptime := totalTime - s.downtime
	return UptimeData{
		Uptime:       float64(uptime) / float64(totalTime) * 100,
		TotalUptime:  totalTime.Round(time.Second),
		LastDowntime: s.lastDowntime,
	}
}

// GetAnomalies returns detected anomalies
func (s *UptimeService) GetAnomalies() []Anomaly {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.anomalies
}

func (s *UptimeService) monitorAnomaly() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.RLock()
		avgResponseTime := s.calculateAvgResponseTime()
		errorRate := float64(s.errorCount) / float64(s.totalRequests)
		s.mu.RUnlock()

		if anomaly := s.anomalyDetector.DetectAnomaly(avgResponseTime, errorRate); anomaly != nil {
			s.mu.Lock()
			s.anomalies = append(s.anomalies, *anomaly)
			if len(s.anomalies) > 10 {
				s.anomalies = s.anomalies[1:]
			}
			s.mu.Unlock()
		}
	}
}

func (s *UptimeService) calculateAvgResponseTime() time.Duration {
	if len(s.responseTimes) == 0 {
		return 0
	}
	var total time.Duration
	for _, rt := range s.responseTimes {
		total += rt
	}
	return total / time.Duration(len(s.responseTimes))
}

// UptimeMiddleware is a middleware that tracks uptime and request statistics
type UptimeMiddleware struct {
	service *UptimeService
}

// NewUptimeMiddleware creates a new UptimeMiddleware
func NewUptimeMiddleware(service *UptimeService) *UptimeMiddleware {
	return &UptimeMiddleware{service: service}
}

// Middleware wraps an http.Handler and records request data
func (m *UptimeMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom ResponseWriter to capture the status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Defer function to handle panics and record request data
		defer func() {
			if err := recover(); err != nil {
				rw.statusCode = http.StatusInternalServerError
				// Log the error and stack trace
				fmt.Printf("panic: %v\n%s", err, debug.Stack())
				// You might want to send a 500 Internal Server Error response here
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}

			duration := time.Since(start)
			isError := rw.statusCode >= 400
			m.service.RecordRequest(duration, isError)
		}()

		// Call the next handler
		next.ServeHTTP(rw, r)
	})
}

// responseWriter is a custom ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// UptimeData represents uptime information
type UptimeData struct {
	Uptime       float64
	TotalUptime  time.Duration
	LastDowntime time.Time
}

// UptimeResponse is the JSON response structure for uptime information
type UptimeResponse struct {
	Uptime           float64   `json:"uptime"`
	UptimePercentage string    `json:"uptimePercentage"`
	Description      string    `json:"description"`
	Status           string    `json:"status"`
	TotalUptime      string    `json:"totalUptime"`
	LastDowntime     string    `json:"lastDowntime"`
	Anomalies        []Anomaly `json:"anomalies"`
}

// Anomaly represents an detected anomaly
type Anomaly struct {
	Timestamp   time.Time `json:"timestamp"`
	Description string    `json:"description"`
}

// AnomalyDetector detects anomalies in system performance
type AnomalyDetector struct {
	avgResponseTimeThreshold time.Duration
	errorRateThreshold       float64
}

// NewAnomalyDetector creates a new AnomalyDetector
func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		avgResponseTimeThreshold: 500 * time.Millisecond,
		errorRateThreshold:       0.05, // 5%
	}
}

// DetectAnomaly checks for anomalies based on average response time and error rate
func (ad *AnomalyDetector) DetectAnomaly(avgResponseTime time.Duration, errorRate float64) *Anomaly {
	if avgResponseTime > ad.avgResponseTimeThreshold {
		return &Anomaly{
			Timestamp:   time.Now(),
			Description: fmt.Sprintf("High average response time: %v", avgResponseTime),
		}
	}
	if errorRate > ad.errorRateThreshold {
		return &Anomaly{
			Timestamp:   time.Now(),
			Description: fmt.Sprintf("High error rate: %.2f%%", errorRate*100),
		}
	}
	return nil
}

func getUptimeDescription(uptime float64) string {
	switch {
	case uptime >= 99.99:
		return "Rock-solid reliability"
	case uptime >= 99.9:
		return "Excellent uptime"
	case uptime >= 99.5:
		return "Very good uptime"
	case uptime >= 99.0:
		return "Good uptime"
	default:
		return "Needs improvement"
	}
}

func getUptimeStatus(uptime float64) string {
	switch {
	case uptime >= 99.99:
		return "Exceptional"
	case uptime >= 99.9:
		return "Excellent"
	case uptime >= 99.5:
		return "Very Good"
	case uptime >= 99.0:
		return "Good"
	default:
		return "Fair"
	}
}
