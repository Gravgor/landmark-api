package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Landmark struct {
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey" json:"-"`
	Name        string          `gorm:"type:varchar(255);not null" json:"name"`
	Description string          `gorm:"type:text;not null" json:"description"`
	Latitude    float64         `gorm:"type:decimal(10,8);not null" json:"latitude"`
	Longitude   float64         `gorm:"type:decimal(11,8);not null" json:"longitude"`
	Country     string          `gorm:"type:varchar(100);not null" json:"country"`
	City        string          `gorm:"type:varchar(100);not null" json:"city"`
	Category    string          `gorm:"type:varchar(50);not null" json:"category"`
	ImageUrl    string          `gorm:"type:varchar(255)" json:"image_url"`
	Images      []LandmarkImage `gorm:"foreignKey:LandmarkID" json:"images"`
	CreatedAt   time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time       `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt   gorm.DeletedAt  `gorm:"index" json:"-"`
}

type LandmarkImage struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"-"`
	LandmarkID uuid.UUID `gorm:"type:uuid;not null" json:"-"`
	ImageURL   string    `gorm:"type:varchar(500);not null" json:"image_url"`
	CreatedAt  time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt  time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

type LandmarkDetail struct {
	ID                     uuid.UUID         `gorm:"type:uuid;primaryKey" json:"-"`
	LandmarkID             uuid.UUID         `gorm:"type:uuid;not null;uniqueIndex" json:"-"`
	OpeningHours           map[string]string `gorm:"type:jsonb" json:"opening_hours"`
	TicketPrices           map[string]string `gorm:"type:jsonb" json:"ticket_prices"`
	HistoricalSignificance string            `gorm:"type:text" json:"historical_significance"`
	VisitorTips            string            `gorm:"type:text" json:"visitor_tips"`
	AccessibilityInfo      string            `gorm:"type:text" json:"accessibility_info"`
	CreatedAt              time.Time         `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt              time.Time         `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt              gorm.DeletedAt    `gorm:"index" json:"-"`
}

type SubmissionLandmark struct {
	ID          uuid.UUID                 `gorm:"type:uuid;primaryKey" json:"-"`
	Name        string                    `gorm:"type:varchar(255);not null" json:"name"`
	Description string                    `gorm:"type:text;not null" json:"description"`
	Latitude    float64                   `gorm:"type:decimal(10,8);not null" json:"latitude"`
	Longitude   float64                   `gorm:"type:decimal(11,8);not null" json:"longitude"`
	Country     string                    `gorm:"type:varchar(100);not null" json:"country"`
	City        string                    `gorm:"type:varchar(100);not null" json:"city"`
	Category    string                    `gorm:"type:varchar(50);not null" json:"category"`
	Status      string                    // "pending", "approved", or "rejected"
	Images      []SubmissionLandmarkImage `gorm:"foreignKey:SubmissionLandmarkID" json:"images"`
	Detail      SubmissionLandmarkDetail  `gorm:"foreignKey:SubmissionLandmarkID;references:ID" json:"details"`
	CreatedAt   time.Time                 `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time                 `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

type SubmissionLandmarkImage struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey" json:"-"`
	SubmissionLandmarkID uuid.UUID `gorm:"type:uuid;not null" json:"-"`
	ImageURL             string    `gorm:"type:varchar(500);not null" json:"image_url"`
	CreatedAt            time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt            time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

type SubmissionLandmarkDetail struct {
	ID                     uuid.UUID `gorm:"type:uuid;primaryKey" json:"-"`
	SubmissionLandmarkID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"-"`
	OpeningHours           JSON      `gorm:"type:jsonb" json:"opening_hours"`
	TicketPrices           JSON      `gorm:"type:jsonb" json:"ticket_prices"`
	HistoricalSignificance string    `gorm:"type:text" json:"historical_significance"`
	VisitorTips            string    `gorm:"type:text" json:"visitor_tips"`
	AccessibilityInfo      string    `gorm:"type:text" json:"accessibility_info"`
	CreatedAt              time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt              time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
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

func (l *Landmark) GetMainImage() string {
	if len(l.Images) > 0 {
		return l.Images[0].ImageURL
	}
	return l.ImageUrl
}

// AddImage adds a new image URL to the Landmark
func (l *Landmark) AddImage(imageURL string) {
	l.Images = append(l.Images, LandmarkImage{
		ID:         uuid.New(),
		LandmarkID: l.ID,
		ImageURL:   imageURL,
	})
}

// BeforeSave GORM hook to ensure data consistency
func (l *Landmark) BeforeSave(tx *gorm.DB) error {
	if len(l.Images) > 0 && l.ImageUrl == "" {
		l.ImageUrl = l.Images[0].ImageURL
	} else if l.ImageUrl != "" && len(l.Images) == 0 {
		l.AddImage(l.ImageUrl)
	}
	return nil
}
