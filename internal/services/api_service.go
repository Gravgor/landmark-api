package services

import (
	"context"
	"landmark-api/internal/models"
	"landmark-api/internal/repository"
	"time"

	"github.com/google/uuid"
)

type APIKeyService interface {
	GenerateAPIKey() string
	AssignAPIKeyToUser(ctx context.Context, userID uuid.UUID) (*models.APIKey, error)
	GetAPIKeyByKey(ctx context.Context, key string) (*models.APIKey, error)
	GetAPIKeyByUserID(ctx context.Context, userID uuid.UUID) (*models.APIKey, error)
	UpdateAPIKey(ctx context.Context, userID uuid.UUID, newKey string) error
	DeleteAPIKey(ctx context.Context, userID uuid.UUID) error
}

type apiKeyService struct {
	apiKeyRepo repository.APIKeyRepository
}

func NewAPIKeyService(apiKeyRepo repository.APIKeyRepository) APIKeyService {
	return &apiKeyService{
		apiKeyRepo: apiKeyRepo,
	}
}

func (s *apiKeyService) GenerateAPIKey() string {
	return uuid.NewString() // Generate a new UUID as the API key
}

func (s *apiKeyService) AssignAPIKeyToUser(ctx context.Context, userID uuid.UUID) (*models.APIKey, error) {
	apiKey := &models.APIKey{
		ID:        uuid.New(),
		UserID:    userID,
		Key:       s.GenerateAPIKey(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.apiKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, err
	}

	return apiKey, nil
}

func (s *apiKeyService) GetAPIKeyByKey(ctx context.Context, key string) (*models.APIKey, error) {
	apiKey, err := s.apiKeyRepo.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	return apiKey, nil
}

func (s *apiKeyService) GetAPIKeyByUserID(ctx context.Context, userID uuid.UUID) (*models.APIKey, error) {
	apiKey, err := s.apiKeyRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return apiKey, nil
}

func (s *apiKeyService) UpdateAPIKey(ctx context.Context, userID uuid.UUID, newKey string) error {
	if err := s.apiKeyRepo.UpdateAPIKey(ctx, userID, newKey); err != nil {
		return err
	}
	return nil
}

func (s *apiKeyService) DeleteAPIKey(ctx context.Context, userID uuid.UUID) error {
	if err := s.apiKeyRepo.DeleteByUserID(ctx, userID); err != nil {
		return err
	}
	return nil
}
