package services

import (
	"context"
	"landmark-api/internal/models"
	"landmark-api/internal/repository"
	"time"
)

type AuditLogService interface {
	GetAuditLogs(ctx context.Context, page, pageSize int) ([]models.AuditLog, int64, error)
	CreateAuditLog(ctx context.Context, adminID int, action, entityType, entityID, details string) error
}

type auditLogService struct {
	auditLogRepo repository.AuditLogRepository
}

func NewAuditLogService(auditLogRepo repository.AuditLogRepository) AuditLogService {
	return &auditLogService{
		auditLogRepo: auditLogRepo,
	}
}

func (s *auditLogService) GetAuditLogs(ctx context.Context, page, pageSize int) ([]models.AuditLog, int64, error) {
	return s.auditLogRepo.ListAuditLogs(ctx, page, pageSize)
}

func (s *auditLogService) CreateAuditLog(ctx context.Context, adminID int, action, entityType, entityID, details string) error {
	log := &models.AuditLog{
		AdminID:    adminID,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Details:    details,
		Timestamp:  time.Now(),
	}
	return s.auditLogRepo.CreateAuditLog(ctx, log)
}
