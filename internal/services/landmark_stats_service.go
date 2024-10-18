package services

import (
	"context"
	"landmark-api/internal/models"
	"landmark-api/internal/repository"
)

type LandmarkStatsService interface {
	GetLandmarkStats(ctx context.Context) (*models.LandmarkStats, error)
}

type landmarkStatsService struct {
	landmarkStatsRepo repository.LandmarkStatsRepository
}

func NewLandmarkStatsService(landmarkStatsRepo repository.LandmarkStatsRepository) LandmarkStatsService {
	return &landmarkStatsService{
		landmarkStatsRepo: landmarkStatsRepo,
	}
}

func (s *landmarkStatsService) GetLandmarkStats(ctx context.Context) (*models.LandmarkStats, error) {
	totalLandmarks, err := s.landmarkStatsRepo.GetTotalLandmarks(ctx)
	if err != nil {
		return nil, err
	}

	landmarksByCategory, err := s.landmarkStatsRepo.GetLandmarksByCategory(ctx)
	if err != nil {
		return nil, err
	}

	landmarksByCountry, err := s.landmarkStatsRepo.GetLandmarksByCountry(ctx)
	if err != nil {
		return nil, err
	}

	recentlyAdded, err := s.landmarkStatsRepo.GetRecentlyAddedLandmarks(ctx, 5) // Get 5 most recent landmarks
	if err != nil {
		return nil, err
	}

	return &models.LandmarkStats{
		TotalLandmarks:      totalLandmarks,
		LandmarksByCategory: landmarksByCategory,
		LandmarksByCountry:  landmarksByCountry,
		RecentlyAdded:       recentlyAdded,
	}, nil
}
