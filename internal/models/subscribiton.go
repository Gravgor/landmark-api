package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubscriptionPlan string

const (
	FreePlan       SubscriptionPlan = "FREE"
	ProPlan        SubscriptionPlan = "PRO"
	EnterprisePlan SubscriptionPlan = "ENTERPRISE"
)

type Subscription struct {
	ID        uuid.UUID        `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID        `gorm:"type:uuid;not null;index" json:"user_id"`
	PlanType  SubscriptionPlan `gorm:"type:varchar(20);not null" json:"plan_type"`
	StartDate time.Time        `gorm:"not null" json:"start_date"`
	EndDate   *time.Time       `gorm:"default:null" json:"end_date"`
	Status    string           `gorm:"type:varchar(50);not null" json:"status"`
	CreatedAt time.Time        `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time        `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt gorm.DeletedAt   `gorm:"index" json:"-"`
	User      User             `gorm:"foreignKey:UserID" json:"-"`
}

func (Subscription) TableName() string {
	return "subscriptions"
}

func (s *Subscription) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}

	now := time.Now()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	if s.UpdatedAt.IsZero() {
		s.UpdatedAt = now
	}

	return nil
}

func (s *Subscription) BeforeUpdate(tx *gorm.DB) error {
	s.UpdatedAt = time.Now()
	return nil
}
