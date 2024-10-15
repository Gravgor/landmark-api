package services

import (
	"landmark-api/internal/models"
	"landmark-api/internal/repository"
	"time"
)

type RequestLogService interface {
	LogRequest(userID, endpoint, method string, statusCode int, status models.RequestStatus, summary string) error
	GetUserLogs(userID string, from, to time.Time) ([]models.RequestLog, error)
	GetEndpointLogs(endpoint string, from, to time.Time) ([]models.RequestLog, error)
}

type requestLogService struct {
	repo repository.RequestLogRepository
}

func NewRequestLogService(repo repository.RequestLogRepository) RequestLogService {
	return &requestLogService{repo: repo}
}

func (s *requestLogService) LogRequest(userID, endpoint, method string, statusCode int, status models.RequestStatus, summary string) error {
	log := &models.RequestLog{
		UserID:     userID,
		Endpoint:   endpoint,
		Method:     method,
		Status:     status,
		StatusCode: statusCode,
		Summary:    summary,
		Timestamp:  time.Now(),
	}
	return s.repo.Create(log)
}

func (s *requestLogService) GetUserLogs(userID string, from, to time.Time) ([]models.RequestLog, error) {
	return s.repo.GetUserLogs(userID, from, to)
}

func (s *requestLogService) GetEndpointLogs(endpoint string, from, to time.Time) ([]models.RequestLog, error) {
	return s.repo.GetEndpointLogs(endpoint, from, to)
}
