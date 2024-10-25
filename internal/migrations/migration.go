// internal/migrations/landmarks.go

package migrations

/*
import (
	"landmark-api/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func MigrateLandmarks(db *gorm.DB) error {
	// First, migrate the schema
	err := db.AutoMigrate(&models.Landmark{}, &models.LandmarkDetail{})
	if err != nil {
		return err
	}

	// Check if we already have landmarks
	var count int64
	db.Model(&models.Landmark{}).Count(&count)
	if count > 0 {
		return nil // Skip if landmarks already exist
	}

	// Create landmarks with their details
	landmarks := []struct {
		landmark models.Landmark
		details  models.LandmarkDetail
	}{
		{
			landmark: models.Landmark{
				ID:          uuid.New(),
				Name:        "Eiffel Tower",
				Description: "Iconic iron lattice tower located on the Champ de Mars in Paris. Built in 1889, it has become both a global cultural icon of France and one of the most recognizable structures in the world.",
				Latitude:    48.8584,
				Longitude:   2.2945,
				Country:     "France",
				City:        "Paris",
				Category:    "Architecture",
				ImageUrl:    "https://images.example.com/eiffel-tower.jpg",
			},
			details: models.LandmarkDetail{
				ID: uuid.New(),
				OpeningHours: map[string]string{
					"Monday-Sunday": "09:00-00:45",
				},
				TicketPrices: map[string]float64{
					"Adult":    26.10,
					"Youth":    13.10,
					"Children": 6.60,
				},
				HistoricalSignificance: "Built for the 1889 World's Fair, the Eiffel Tower commemorated the centennial of the French Revolution.",
				VisitorTips:            "Book tickets online in advance to avoid long queues. Visit during sunset for spectacular views.",
				AccessibilityInfo:      "Accessible by elevator to the second floor. Wheelchair access available.",
			},
		},
		{
			landmark: models.Landmark{
				ID:          uuid.New(),
				Name:        "Great Wall of China",
				Description: "Ancient series of walls and fortifications, built across the historical northern borders of ancient Chinese states and Imperial China as protection.",
				Latitude:    40.4319,
				Longitude:   116.5704,
				Country:     "China",
				City:        "Beijing",
				Category:    "Historical",
				ImageUrl:    "https://images.example.com/great-wall.jpg",
			},
			details: models.LandmarkDetail{
				ID: uuid.New(),
				OpeningHours: map[string]string{
					"Monday-Sunday": "07:30-17:30",
				},
				TicketPrices: map[string]float64{
					"Adult":    45.00,
					"Student":  25.00,
					"Children": 0.00,
				},
				HistoricalSignificance: "Built over many centuries by various dynasties, it's one of the most impressive architectural feats in human history.",
				VisitorTips:            "Visit the Mutianyu section for fewer crowds and better preserved walls. Bring comfortable walking shoes.",
				AccessibilityInfo:      "Some sections have cable cars for easier access.",
			},
		},
		{
			landmark: models.Landmark{
				ID:          uuid.New(),
				Name:        "Taj Mahal",
				Description: "An ivory-white marble mausoleum on the right bank of the river Yamuna, commissioned in 1632 by the Mughal emperor Shah Jahan to house the tomb of his favorite wife.",
				Latitude:    27.1751,
				Longitude:   78.0421,
				Country:     "India",
				City:        "Agra",
				Category:    "Architecture",
				ImageUrl:    "https://images.example.com/taj-mahal.jpg",
			},
			details: models.LandmarkDetail{
				ID: uuid.New(),
				OpeningHours: map[string]string{
					"Monday-Sunday": "06:00-18:30",
					"Friday":        "Closed",
				},
				TicketPrices: map[string]float64{
					"Foreign Tourist": 1100.00,
					"Indian Citizen":  50.00,
					"Children":        0.00,
				},
				HistoricalSignificance: "Built as a testament of love by Shah Jahan for his beloved wife Mumtaz Mahal.",
				VisitorTips:            "Visit during sunrise for the best photographs. Bring shoe covers or be prepared to remove shoes.",
				AccessibilityInfo:      "Wheelchair availability at entrance. Golf cart transport available.",
			},
		},
		{
			landmark: models.Landmark{
				ID:          uuid.New(),
				Name:        "Machu Picchu",
				Description: "15th-century Inca citadel set high in the Andes Mountains, built with sophisticated dry stone walls that fuse huge blocks without mortar.",
				Latitude:    -13.1631,
				Longitude:   -72.5450,
				Country:     "Peru",
				City:        "Cusco Region",
				Category:    "Archaeological",
				ImageUrl:    "https://images.example.com/machu-picchu.jpg",
			},
			details: models.LandmarkDetail{
				ID: uuid.New(),
				OpeningHours: map[string]string{
					"Monday-Sunday": "06:00-17:30",
				},
				TicketPrices: map[string]float64{
					"Foreign Adult": 152.00,
					"Student":       77.00,
					"Children":      70.00,
				},
				HistoricalSignificance: "Ancient Incan city that shows their architectural and agricultural prowess.",
				VisitorTips:            "Book tickets months in advance. Acclimatize to the altitude before visiting.",
				AccessibilityInfo:      "Limited accessibility due to steep terrain and historic preservation.",
			},
		},
		{
			landmark: models.Landmark{
				ID:          uuid.New(),
				Name:        "Petra",
				Description: "Ancient city famous for its rock-cut architecture and water conduit system, established possibly as early as 312 BCE.",
				Latitude:    30.3285,
				Longitude:   35.4444,
				Country:     "Jordan",
				City:        "Ma'an Governorate",
				Category:    "Archaeological",
				ImageUrl:    "https://images.example.com/petra.jpg",
			},
			details: models.LandmarkDetail{
				ID: uuid.New(),
				OpeningHours: map[string]string{
					"Summer": "06:00-18:00",
					"Winter": "06:00-16:00",
				},
				TicketPrices: map[string]float64{
					"One Day":    50.00,
					"Two Days":   55.00,
					"Three Days": 60.00,
				},
				HistoricalSignificance: "Capital of the Nabataean Kingdom, showcasing advanced architectural and engineering capabilities.",
				VisitorTips:            "Start early to avoid heat. Comfortable walking shoes essential.",
				AccessibilityInfo:      "Horse carriages available for transportation within the site.",
			},
		},
	}

	// Create each landmark and its details in a transaction
	for _, item := range landmarks {
		err := db.Transaction(func(tx *gorm.DB) error {
			// Set timestamps
			now := time.Now()
			item.landmark.CreatedAt = now
			item.landmark.UpdatedAt = now
			item.details.CreatedAt = now
			item.details.UpdatedAt = now

			// Create the landmark
			if err := tx.Create(&item.landmark).Error; err != nil {
				return err
			}

			// Set the landmark ID in details
			item.details.LandmarkID = item.landmark.ID

			// Create the details
			if err := tx.Create(&item.details).Error; err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}

// Helper function to create a UUID
func newUUID() uuid.UUID {
	return uuid.New()
}
*/
