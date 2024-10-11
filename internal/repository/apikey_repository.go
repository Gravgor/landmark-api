package repository

import (
	"context"
	"landmark-api/internal/errors"
	"landmark-api/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type APIKeyRepository interface {
	Create(ctx context.Context, apiKey *models.APIKey) error
	GetByKey(ctx context.Context, key string) (*models.APIKey, error)
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	UpdateAPIKey(ctx context.Context, userID uuid.UUID, apiKey string) error
}

type apiKeyRepository struct {
	db *gorm.DB
}

func NewAPIKeyRepository(db *gorm.DB) APIKeyRepository {
	return &apiKeyRepository{db: db}
}

func (r *apiKeyRepository) Create(ctx context.Context, apiKey *models.APIKey) error {
	result := r.db.WithContext(ctx).Create(apiKey)
	if result.Error != nil {
		return errors.Wrap(result.Error, "failed to create API key")
	}
	return nil
}

func (r *apiKeyRepository) GetByKey(ctx context.Context, key string) (*models.APIKey, error) {
	var apiKey models.APIKey
	result := r.db.WithContext(ctx).First(&apiKey, "key = ?", key)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, errors.ErrNotFound
		}
		return nil, errors.Wrap(result.Error, "failed to get API key by key")
	}

	return &apiKey, nil
}

func (r *apiKeyRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.APIKey{}, "user_id = ?", userID)

	if result.Error != nil {
		return errors.Wrap(result.Error, "failed to delete API key")
	}

	if result.RowsAffected == 0 {
		return errors.ErrNotFound
	}

	return nil
}

func (r *apiKeyRepository) UpdateAPIKey(ctx context.Context, userID uuid.UUID, apiKey string) error {
	result := r.db.WithContext(ctx).Model(&models.APIKey{}).Where("user_id = ?", userID).Updates(map[string]interface{}{
		"key":        apiKey,
		"updated_at": time.Now(),
	})

	if result.Error != nil {
		return errors.Wrap(result.Error, "failed to update API key")
	}

	if result.RowsAffected == 0 {
		return errors.ErrNotFound // No API key found for this user
	}

	return nil
}
