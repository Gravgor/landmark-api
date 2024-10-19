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
	GetUserAndSubscriptionByAPIKey(ctx context.Context, key string) (*models.User, *models.Subscription, error)
	GetAPIKeyByUserID(ctx context.Context, userID uuid.UUID) (*models.APIKey, error)
	UpdateAPIKey(ctx context.Context, userID uuid.UUID, newKey string) error
	DeleteAPIKey(ctx context.Context, userID uuid.UUID) error
}

type apiKeyService struct {
	apiKeyRepo repository.APIKeyRepository
	userRepo   repository.UserRepository
	subRepo    repository.SubscriptionRepository
}

func NewAPIKeyService(apiKeyRepo repository.APIKeyRepository, userRepo repository.UserRepository, subRepo repository.SubscriptionRepository) APIKeyService {
	return &apiKeyService{
		apiKeyRepo: apiKeyRepo,
		userRepo:   userRepo,
		subRepo:    subRepo,
	}
}

func (s *apiKeyService) GenerateAPIKey() string {
	return uuid.NewString()
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
	return s.apiKeyRepo.GetByKey(ctx, key)
}

func (s *apiKeyService) GetUserAndSubscriptionByAPIKey(ctx context.Context, key string) (*models.User, *models.Subscription, error) {
	apiKey, err := s.apiKeyRepo.GetByKey(ctx, key)
	if err != nil {
		return nil, nil, err
	}

	user, err := s.userRepo.GetByID(ctx, apiKey.UserID)
	if err != nil {
		return nil, nil, err
	}

	subscription, err := s.subRepo.GetActiveByUserID(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user, subscription, nil
}

func (s *apiKeyService) GetAPIKeyByUserID(ctx context.Context, userID uuid.UUID) (*models.APIKey, error) {
	return s.apiKeyRepo.GetByUserID(ctx, userID)
}

func (s *apiKeyService) UpdateAPIKey(ctx context.Context, userID uuid.UUID, newKey string) error {
	return s.apiKeyRepo.UpdateAPIKey(ctx, userID, newKey)
}

func (s *apiKeyService) DeleteAPIKey(ctx context.Context, userID uuid.UUID) error {
	return s.apiKeyRepo.DeleteByUserID(ctx, userID)
}
