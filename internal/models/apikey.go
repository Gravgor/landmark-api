package models

import (
	"time"

	"github.com/google/uuid"
)

type APIKey struct {
	ID        uuid.UUID `gorm:"type:uuid" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid" json:"user_id"`
	Key       string    `json:"key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
