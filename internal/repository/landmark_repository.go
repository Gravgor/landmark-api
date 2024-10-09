package repository

import (
	"context"
	"encoding/json"
	"errors"
	"landmark-api/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LandmarkRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.Landmark, error)
	List(ctx context.Context, limit, offset int) ([]models.Landmark, error)
	Create(ctx context.Context, landmark *models.Landmark) error
	Update(ctx context.Context, landmark *models.Landmark) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetDetails(ctx context.Context, id uuid.UUID) (*models.LandmarkDetail, error)
	FindByCountry(ctx context.Context, country string) ([]models.Landmark, error)
	FindByName(ctx context.Context, name string) ([]models.Landmark, error)
}

type landmarkRepository struct {
	db *gorm.DB
}

func NewLandmarkRepository(db *gorm.DB) LandmarkRepository {
	return &landmarkRepository{db: db}
}

func (r *landmarkRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Landmark, error) {
	var landmark models.Landmark

	err := r.db.WithContext(ctx).First(&landmark, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &landmark, err
}

func (r *landmarkRepository) List(ctx context.Context, limit, offset int) ([]models.Landmark, error) {
	var landmarks []models.Landmark

	err := r.db.WithContext(ctx).Limit(limit).Offset(offset).
		Order("created_at DESC").Find(&landmarks).Error

	return landmarks, err
}

func (r *landmarkRepository) Create(ctx context.Context, landmark *models.Landmark) error {
	return r.db.WithContext(ctx).Create(landmark).Error
}

func (r *landmarkRepository) Update(ctx context.Context, landmark *models.Landmark) error {
	err := r.db.WithContext(ctx).Model(&models.Landmark{}).
		Where("id = ?", landmark.ID).
		Updates(models.Landmark{
			Name:        landmark.Name,
			Description: landmark.Description,
			Latitude:    landmark.Latitude,
			Longitude:   landmark.Longitude,
			Country:     landmark.Country,
			City:        landmark.City,
			Category:    landmark.Category,
		}).Error

	return err
}

func (r *landmarkRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Landmark{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *landmarkRepository) GetDetails(ctx context.Context, id uuid.UUID) (*models.LandmarkDetail, error) {
	var detail models.LandmarkDetail
	var openingHoursJSON, ticketPricesJSON string

	err := r.db.WithContext(ctx).
		Table("landmark_details").
		Select("id, landmark_id, opening_hours, ticket_prices, historical_significance, visitor_tips, accessibility_info, created_at, updated_at").
		Where("landmark_id = ?", id).
		Row().Scan(&detail.ID, &detail.LandmarkID, &openingHoursJSON, &ticketPricesJSON,
		&detail.HistoricalSignificance, &detail.VisitorTips, &detail.AccessibilityInfo,
		&detail.CreatedAt, &detail.UpdatedAt)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON data
	if err := json.Unmarshal([]byte(openingHoursJSON), &detail.OpeningHours); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(ticketPricesJSON), &detail.TicketPrices); err != nil {
		return nil, err
	}

	return &detail, nil
}

// FindByCountry retrieves landmarks by country from the database.
func (r *landmarkRepository) FindByCountry(ctx context.Context, country string) ([]models.Landmark, error) {
	var landmarks []models.Landmark

	err := r.db.WithContext(ctx).
		Where("country = ?", country).
		Order("created_at DESC").
		Find(&landmarks).Error
	return landmarks, err
}

// FindByName retrieves landmarks by name from the database.
func (r *landmarkRepository) FindByName(ctx context.Context, name string) ([]models.Landmark, error) {
	var landmarks []models.Landmark

	err := r.db.WithContext(ctx).
		Where("name ILIKE ?", "%"+name+"%").
		Order("created_at DESC").
		Find(&landmarks).Error

	return landmarks, err
}
