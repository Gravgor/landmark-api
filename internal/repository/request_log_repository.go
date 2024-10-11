package repository

import (
	"landmark-api/internal/models"
	"time"

	"gorm.io/gorm"
)

type RequestLogRepository interface {
	Create(log *models.RequestLog) error
	GetUserLogs(userID string, from, to time.Time) ([]models.RequestLog, error)
	GetEndpointLogs(endpoint string, from, to time.Time) ([]models.RequestLog, error)
}

type requestLogRepository struct {
	db *gorm.DB
}

func NewRequestLogRepository(db *gorm.DB) RequestLogRepository {
	return &requestLogRepository{db: db}
}

func (r *requestLogRepository) Create(log *models.RequestLog) error {
	return r.db.Create(log).Error
}

func (r *requestLogRepository) GetUserLogs(userID string, from, to time.Time) ([]models.RequestLog, error) {
	var logs []models.RequestLog
	err := r.db.Where("user_id = ? AND timestamp BETWEEN ? AND ?", userID, from, to).
		Order("timestamp desc").
		Find(&logs).Error
	return logs, err
}

func (r *requestLogRepository) GetEndpointLogs(endpoint string, from, to time.Time) ([]models.RequestLog, error) {
	var logs []models.RequestLog
	err := r.db.Where("endpoint = ? AND timestamp BETWEEN ? AND ?", endpoint, from, to).
		Order("timestamp desc").
		Find(&logs).Error
	return logs, err
}
