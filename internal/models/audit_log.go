package models

import (
	"time"

	"gorm.io/gorm"
)

type AuditLog struct {
	gorm.Model
	AdminID    int       `json:"adminId"`
	Action     string    `json:"action"`
	EntityType string    `json:"entityType"`
	EntityID   string    `json:"entityId"`
	Details    string    `json:"details"`
	Timestamp  time.Time `json:"timestamp"`
}
