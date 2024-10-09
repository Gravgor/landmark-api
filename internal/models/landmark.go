package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Landmark struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string         `gorm:"type:varchar(255);not null" json:"name"`
	Description string         `gorm:"type:text;not null" json:"description"`
	Latitude    float64        `gorm:"type:decimal(10,8);not null" json:"latitude"`
	Longitude   float64        `gorm:"type:decimal(11,8);not null" json:"longitude"`
	Country     string         `gorm:"type:varchar(100);not null" json:"country"`
	City        string         `gorm:"type:varchar(100);not null" json:"city"`
	Category    string         `gorm:"type:varchar(50);not null" json:"category"`
	CreatedAt   time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type LandmarkDetail struct {
	ID                     uuid.UUID          `gorm:"type:uuid;primaryKey" json:"id"`
	LandmarkID             uuid.UUID          `gorm:"type:uuid;not null;uniqueIndex" json:"landmark_id"`
	OpeningHours           map[string]string  `gorm:"type:jsonb" json:"opening_hours"`
	TicketPrices           map[string]float64 `gorm:"type:jsonb" json:"ticket_prices"`
	HistoricalSignificance string             `gorm:"type:text" json:"historical_significance"`
	VisitorTips            string             `gorm:"type:text" json:"visitor_tips"`
	AccessibilityInfo      string             `gorm:"type:text" json:"accessibility_info"`
	CreatedAt              time.Time          `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt              time.Time          `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt              gorm.DeletedAt     `gorm:"index" json:"-"`
}

func (Landmark) TableName() string {
	return "landmarks"
}

func (LandmarkDetail) TableName() string {
	return "landmark_details"
}

func (l *Landmark) BeforeCreate(tx *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	now := time.Now()
	if l.CreatedAt.IsZero() {
		l.CreatedAt = now
	}
	if l.UpdatedAt.IsZero() {
		l.UpdatedAt = now
	}
	return nil
}

func (l *Landmark) BeforeUpdate(tx *gorm.DB) error {
	l.UpdatedAt = time.Now()
	return nil
}

func (ld *LandmarkDetail) BeforeCreate(tx *gorm.DB) error {
	if ld.ID == uuid.Nil {
		ld.ID = uuid.New()
	}
	now := time.Now()
	if ld.CreatedAt.IsZero() {
		ld.CreatedAt = now
	}
	if ld.UpdatedAt.IsZero() {
		ld.UpdatedAt = now
	}
	return nil
}

func (ld *LandmarkDetail) BeforeUpdate(tx *gorm.DB) error {
	ld.UpdatedAt = time.Now()
	return nil
}
