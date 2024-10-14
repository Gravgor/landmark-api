package models

import (
	"time"

	"gorm.io/gorm"
)

type APIUsage struct {
	ID           uint   `gorm:"primarykey"`
	UserID       string `gorm:"index"`
	RequestCount int
	PeriodStart  time.Time `gorm:"index"`
	PeriodEnd    time.Time `gorm:"index"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}
