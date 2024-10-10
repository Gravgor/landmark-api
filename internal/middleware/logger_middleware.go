package middleware

import (
	"landmark-api/cmd/logger"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// LoggingMiddleware logs the details of each request and response
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer to capture the status code
		rw := &responseWriter{w, http.StatusOK}

		// Call the next handler
		next.ServeHTTP(rw, r)

		// Log request details
		logger.LogEvent(logrus.InfoLevel, "Request handled", logrus.Fields{
			"method":        r.Method,
			"url":           r.URL.Path,
			"status_code":   rw.statusCode,
			"response_time": time.Since(start).Milliseconds(),
			"ip":            r.RemoteAddr,
		})
	})
}

// responseWriter is a wrapper around http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
