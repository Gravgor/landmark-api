package services

import (
	"context"
	"landmark-api/internal/errors"
	"landmark-api/internal/models"
	"landmark-api/internal/repository"

	"github.com/google/uuid"
)

type LandmarkService interface {
	GetLandmark(ctx context.Context, id uuid.UUID) (*models.Landmark, error)
	ListLandmarks(ctx context.Context, page, pageSize int) ([]models.Landmark, error)
	GetLandmarksWithFilters(ctx context.Context, page, perPage int, searchTerm, category string) ([]models.Landmark, int64, error)
	GetLandmarkDetails(ctx context.Context, id uuid.UUID, userSubscription models.SubscriptionPlan) (*models.LandmarkDetail, error)
	GetLandmarkAdminDetails(ctx context.Context, id uuid.UUID) (*models.LandmarkDetail, error)
	GetLandmarksByCountry(ctx context.Context, country string) ([]models.Landmark, error)
	GetLandmarksByName(ctx context.Context, name string) ([]models.Landmark, error)
}

type landmarkService struct {
	landmarkRepo repository.LandmarkRepository
}

func NewLandmarkService(landmarkRepo repository.LandmarkRepository) LandmarkService {
	return &landmarkService{landmarkRepo: landmarkRepo}
}

func (s *landmarkService) GetLandmark(ctx context.Context, id uuid.UUID) (*models.Landmark, error) {
	return s.landmarkRepo.GetByID(ctx, id)
}

func (s *landmarkService) GetLandmarksWithFilters(ctx context.Context, page, perPage int, searchTerm, category string) ([]models.Landmark, int64, error) {
	return s.landmarkRepo.ListWithFilters(ctx, page, perPage, searchTerm, category)
}

func (s *landmarkService) ListLandmarks(ctx context.Context, page, pageSize int) ([]models.Landmark, error) {
	offset := (page - 1) * pageSize
	return s.landmarkRepo.List(ctx, pageSize, offset)
}

func (s *landmarkService) GetLandmarkDetails(ctx context.Context, id uuid.UUID, userSubscription models.SubscriptionPlan) (*models.LandmarkDetail, error) {
	if userSubscription == models.FreePlan {
		return nil, errors.ErrInsufficientSubscription
	}
	return s.landmarkRepo.GetDetails(ctx, id)
}

func (s *landmarkService) GetLandmarkAdminDetails(ctx context.Context, id uuid.UUID) (*models.LandmarkDetail, error) {
	return s.landmarkRepo.GetDetails(ctx, id)
}

// GetLandmarksByCountry retrieves landmarks by country from the repository.
func (s *landmarkService) GetLandmarksByCountry(ctx context.Context, country string) ([]models.Landmark, error) {
	return s.landmarkRepo.FindByCountry(ctx, country)
}

// GetLandmarksByName retrieves landmarks by name from the repository.
func (s *landmarkService) GetLandmarksByName(ctx context.Context, name string) ([]models.Landmark, error) {
	return s.landmarkRepo.FindByName(ctx, name)
}
