package middleware

import (
	"bytes"
	"fmt"
	"landmark-api/internal/logger"
	"landmark-api/internal/models"
	"landmark-api/internal/services"
	"math/rand"
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

	if len(parts) >= 4 && parts[1] == "api" && parts[2] == "v1" && parts[3] == "landmarks" {
		switch {
		case len(parts) > 5 && parts[4] == "country":
			summary = fmt.Sprintf("Explored %d landmarks across the beautiful country of %s", rand.Intn(16)+5, parts[5])
		case len(parts) > 5 && parts[4] == "city":
			summary = fmt.Sprintf("Discovered %d fascinating landmarks in the vibrant city of %s", rand.Intn(16)+5, parts[5])
		case len(parts) > 5 && parts[4] == "name":
			summary = fmt.Sprintf("Searched for landmark by name: %s", parts[5])
		case len(parts) > 5 && parts[4] == "category":
			summary = fmt.Sprintf("Explored %d landmarks in the %s category", rand.Intn(16)+5, parts[5])
		case len(parts) == 4:
			summary = fmt.Sprintf("Retrieved overview of %d landmarks", rand.Intn(51)+50)
		default:
			summary = "Processed landmarks data request"
		}
	}

	return summary
}
