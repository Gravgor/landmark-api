package repository

import (
	"context"
	"landmark-api/internal/models"

	"gorm.io/gorm"
)

type AuditLogRepository interface {
	ListAuditLogs(ctx context.Context, page, pageSize int) ([]models.AuditLog, int64, error)
	CreateAuditLog(ctx context.Context, log *models.AuditLog) error
}

type auditLogRepository struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &auditLogRepository{
		db: db,
	}
}

func (r *auditLogRepository) ListAuditLogs(ctx context.Context, page, pageSize int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	offset := (page - 1) * pageSize

	err := r.db.WithContext(ctx).Model(&models.AuditLog{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = r.db.WithContext(ctx).
		Order("timestamp DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&logs).Error

	return logs, total, err
}

func (r *auditLogRepository) CreateAuditLog(ctx context.Context, log *models.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}
