package middleware

import (
	"bytes"
	"landmark-api/internal/logger"
	"landmark-api/internal/models"
	"landmark-api/internal/services"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

type ResponseWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (rw *ResponseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *ResponseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

type RequestLogger struct {
	logService services.RequestLogService
}

func NewRequestLogger(logService services.RequestLogService) *RequestLogger {
	return &RequestLogger{
		logService: logService,
	}
}

func (rl *RequestLogger) LogRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create custom response writer to capture status code
		rw := &ResponseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		// Get user from context
		user, ok := services.UserFromContext(r.Context())
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		// Create summary based on the endpoint
		summary := createRequestSummary(r)

		// Execute the request
		next.ServeHTTP(rw, r)

		// Determine status
		status := models.StatusSuccess
		if rw.status >= 400 {
			status = models.StatusError
		}

		// Log to database
		err := rl.logService.LogRequest(
			user.ID.String(),
			r.URL.Path,
			r.Method,
			rw.status,
			status,
			summary,
		)

		if err != nil {
			logger.Logger.WithFields(logrus.Fields{
				"error": err,
				"user":  user.ID,
				"path":  r.URL.Path,
			}).Error("Failed to log request")
		}
	})
}

func createRequestSummary(r *http.Request) string {
	parts := strings.Split(r.URL.Path, "/")
	summary := "API request"

	// Example of creating meaningful summaries based on endpoints
	if len(parts) >= 3 && parts[1] == "landmarks" {
		switch parts[2] {
		case "country":
			if len(parts) > 3 {
				summary = "Landmark search for country: " + parts[3]
			}
		case "city":
			if len(parts) > 3 {
				summary = "Landmark search for city: " + parts[3]
			}
		case "name":
			if len(parts) > 3 {
				summary = "Landmark search by name: " + parts[3]
			}
		}
	}

	return summary
}
