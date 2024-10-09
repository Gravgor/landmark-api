package repository

import (
	"context"
	"errors"
	"landmark-api/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubscriptionRepository interface {
	Create(ctx context.Context, subscription *models.Subscription) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Subscription, error)
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*models.Subscription, error)
	Update(ctx context.Context, subscription *models.Subscription) error
	CancelSubscription(ctx context.Context, subscriptionID uuid.UUID) error
	GetSubscriptionHistory(ctx context.Context, userID uuid.UUID) ([]*models.Subscription, error)
}

var (
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrSubscriptionExists   = errors.New("active subscription already exists")
)

type subscriptionRepository struct {
	db *gorm.DB
}

func NewSubscriptionRepository(db *gorm.DB) SubscriptionRepository {
	return &subscriptionRepository{
		db: db,
	}
}

func (r *subscriptionRepository) Create(ctx context.Context, subscription *models.Subscription) error {
	// Check for an active subscription
	existingSub, err := r.GetActiveByUserID(ctx, subscription.UserID)
	if err != nil && !errors.Is(err, ErrSubscriptionNotFound) {
		return err
	}
	if existingSub != nil {
		return ErrSubscriptionExists
	}

	// Create the new subscription
	if err := r.db.WithContext(ctx).Create(subscription).Error; err != nil {
		return err
	}

	return nil
}

func (r *subscriptionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Subscription, error) {
	var subscription models.Subscription

	err := r.db.WithContext(ctx).First(&subscription, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSubscriptionNotFound
	}

	return &subscription, err
}

func (r *subscriptionRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*models.Subscription, error) {
	var subscription models.Subscription

	err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = 'active' AND (end_date IS NULL OR end_date > ?)", userID, time.Now()).
		Order("created_at DESC").
		First(&subscription).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSubscriptionNotFound
	}

	return &subscription, err
}

func (r *subscriptionRepository) Update(ctx context.Context, subscription *models.Subscription) error {
	err := r.db.WithContext(ctx).Model(&models.Subscription{}).
		Where("id = ?", subscription.ID).
		Updates(map[string]interface{}{
			"plan_type":  subscription.PlanType,
			"end_date":   subscription.EndDate,
			"status":     subscription.Status,
			"updated_at": time.Now(),
		}).Error

	if err != nil {
		return err
	}

	// Check if no rows were affected
	if r.db.RowsAffected == 0 {
		return ErrSubscriptionNotFound
	}

	return nil
}

func (r *subscriptionRepository) CancelSubscription(ctx context.Context, subscriptionID uuid.UUID) error {
	err := r.db.WithContext(ctx).Model(&models.Subscription{}).
		Where("id = ? AND status = 'active'", subscriptionID).
		Updates(map[string]interface{}{
			"status":     "cancelled",
			"end_date":   time.Now(),
			"updated_at": time.Now(),
		}).Error

	if err != nil {
		return err
	}

	if r.db.RowsAffected == 0 {
		return ErrSubscriptionNotFound
	}

	return nil
}

func (r *subscriptionRepository) GetSubscriptionHistory(ctx context.Context, userID uuid.UUID) ([]*models.Subscription, error) {
	var subscriptions []*models.Subscription

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&subscriptions).Error

	return subscriptions, err
}
