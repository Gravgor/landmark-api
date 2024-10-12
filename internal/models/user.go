package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Name            string         `gorm:"type:varchar(255);not null" json:"name"`
	Email           string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash    string         `gorm:"type:varchar(255);not null" json:"-"`
	Role            string         `gorm:"type:varchar(255);not null;default:'user'" json:"role"`
	APIKeys         []APIKey       `gorm:"foreignkey:UserID" json:"api_keys,omitempty"` // Add this line
	StripeID        string         `gorm:"type:varchar(255);not null;default:''" json:"stripe_id"`
	HasAccess       bool           `gorm:"type:boolean;not null;default:false" json:"has_access"`
	AccessGrantedAt time.Time      `gorm:"default:null" json:"access_granted_at"`
	AccessRevokedAt time.Time      `gorm:"default:null" json:"access_revoked_at"`
	CreatedAt       time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"` // Adds soft delete capability
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}

	now := time.Now()
	if u.CreatedAt.IsZero() {
		u.CreatedAt = now
	}
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt = now
	}

	return nil
}

func (u *User) BeforeUpdate(tx *gorm.DB) error {
	u.UpdatedAt = time.Now()
	return nil
}

func (User) TableName() string {
	return "users"
}
