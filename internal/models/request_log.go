package models

import (
	"time"

	"gorm.io/gorm"
)

type RequestStatus string

const (
	StatusSuccess RequestStatus = "SUCCESS"
	StatusError   RequestStatus = "ERROR"
)

type RequestLog struct {
	ID         uint   `gorm:"primarykey"`
	UserID     string `gorm:"index"`
	Endpoint   string `gorm:"index"`
	Method     string
	Status     RequestStatus
	StatusCode int
	Summary    string
	Timestamp  time.Time `gorm:"index"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}
