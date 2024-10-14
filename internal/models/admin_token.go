package models

import "time"

type AdminToken struct {
	ID        uint      `gorm:"primarykey"`
	Token     string    `gorm:"uniqueIndex"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}
