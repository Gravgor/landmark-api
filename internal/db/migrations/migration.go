package migrations

import (
	"landmark-api/internal/models"
	"time"

	"gorm.io/gorm"
)

type Migration struct {
	Name string
	Run  func(*gorm.DB) error
}

func GetMigrations() []Migration {
	return []Migration{
		{
			Name: "CreateLandmarksTableAndInsertData",
			Run: func(db *gorm.DB) error {
				// First, create the table
				if err := db.AutoMigrate(&models.Landmark{}); err != nil {
					return err
				}

				// Then, insert the data
				landmarks := []models.Landmark{
					{
						Model: gorm.Model{
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						Name:            "Eiffel Tower",
						Country:         "France",
						City:            "Paris",
						Description:     "The Eiffel Tower is a wrought-iron lattice tower on the Champ de Mars in Paris. Constructed from 1887 to 1889 as the entrance to the 1889 World's Fair, it was initially criticized by some of France's leading artists and intellectuals for its design, but it has become a global cultural icon of France.",
						Height:          324,
						YearBuilt:       1889,
						Architect:       "Gustave Eiffel",
						VisitorsPerYear: 7000000,
						ImageURL:        "/api/placeholder/800/600", // Placeholder for Eiffel Tower
						Latitude:        48.8584,
						Longitude:       2.2945,
					},
					{
						Model: gorm.Model{
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						Name:            "Taj Mahal",
						Country:         "India",
						City:            "Agra",
						Description:     "The Taj Mahal is an ivory-white marble mausoleum on the right bank of the river Yamuna in Agra, India. It was commissioned in 1632 by the Mughal emperor Shah Jahan to house the tomb of his favorite wife, Mumtaz Mahal; it also houses the tomb of Shah Jahan himself.",
						Height:          73,
						YearBuilt:       1653,
						Architect:       "Ustad Ahmad Lahauri",
						VisitorsPerYear: 8000000,
						ImageURL:        "/api/placeholder/800/600", // Placeholder for Taj Mahal
						Latitude:        27.1751,
						Longitude:       78.0421,
					},
					{
						Model: gorm.Model{
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						Name:            "Great Wall of China",
						Country:         "China",
						City:            "Beijing",
						Description:     "The Great Wall of China is a series of fortification systems generally built across the historical northern borders of ancient Chinese states and Imperial China as protection against various nomadic groups from the Eurasian Steppe.",
						Height:          8,
						YearBuilt:       -700, // Represents 700 BCE
						Architect:       "Multiple dynasties",
						VisitorsPerYear: 10000000,
						ImageURL:        "/api/placeholder/800/600", // Placeholder for Great Wall
						Latitude:        40.4319,
						Longitude:       116.5704,
					},
					{
						Model: gorm.Model{
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						Name:            "Machu Picchu",
						Country:         "Peru",
						City:            "Cusco Region",
						Description:     "Machu Picchu is an Incan citadel set high in the Andes Mountains in Peru, above the Urubamba River valley. Built in the 15th century and later abandoned, it's renowned for its sophisticated dry-stone walls that fuse huge blocks without the use of mortar, intriguing buildings that play on astronomical alignments and panoramic views.",
						Height:          2430, // Height above sea level
						YearBuilt:       1450,
						Architect:       "Inca Empire",
						VisitorsPerYear: 1578030,
						ImageURL:        "/api/placeholder/800/600", // Placeholder for Machu Picchu
						Latitude:        -13.1631,
						Longitude:       -72.5450,
					},
					{
						Model: gorm.Model{
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						Name:            "Petra",
						Country:         "Jordan",
						City:            "Ma'an Governorate",
						Description:     "Petra is a famous archaeological site in Jordan's southwestern desert. Dating to around 300 B.C., it was the capital of the Nabatean Kingdom. Accessed via a narrow canyon called Al Siq, it contains tombs and temples carved into pink sandstone cliffs, earning its nickname, the 'Rose City.'",
						Height:          0,    // Not applicable
						YearBuilt:       -312, // Represents 312 BCE
						Architect:       "Nabataeans",
						VisitorsPerYear: 1135300,
						ImageURL:        "/api/placeholder/800/600", // Placeholder for Petra
						Latitude:        30.3285,
						Longitude:       35.4444,
					},
					{
						Model: gorm.Model{
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						Name:            "Angkor Wat",
						Country:         "Cambodia",
						City:            "Siem Reap",
						Description:     "Angkor Wat is a temple complex in Cambodia and is the largest religious monument in the world by land area. Originally constructed as a Hindu temple dedicated to the god Vishnu for the Khmer Empire, it was gradually transformed into a Buddhist temple towards the end of the 12th century.",
						Height:          65,
						YearBuilt:       1113,
						Architect:       "Suryavarman II",
						VisitorsPerYear: 2600000,
						ImageURL:        "/api/placeholder/800/600", // Placeholder for Angkor Wat
						Latitude:        13.4125,
						Longitude:       103.8670,
					},
					{
						Model: gorm.Model{
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						Name:            "Statue of Liberty",
						Country:         "United States",
						City:            "New York City",
						Description:     "The Statue of Liberty is a colossal neoclassical sculpture on Liberty Island in New York Harbor. The copper statue, a gift from the people of France, was designed by French sculptor Frédéric Auguste Bartholdi and its metal framework was built by Gustave Eiffel.",
						Height:          93,
						YearBuilt:       1886,
						Architect:       "Frédéric Auguste Bartholdi",
						VisitorsPerYear: 4200000,
						ImageURL:        "/api/placeholder/800/600", // Placeholder for Statue of Liberty
						Latitude:        40.6892,
						Longitude:       -74.0445,
					},
					{
						Model: gorm.Model{
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						Name:            "Colosseum",
						Country:         "Italy",
						City:            "Rome",
						Description:     "The Colosseum is an oval amphitheatre in the centre of Rome, Italy. Built of travertine limestone, tuff, and brick-faced concrete, it was the largest amphitheatre ever built at the time and held 50,000 to 80,000 spectators.",
						Height:          48,
						YearBuilt:       80,
						Architect:       "Vespasian",
						VisitorsPerYear: 7400000,
						ImageURL:        "/api/placeholder/800/600", // Placeholder for Colosseum
						Latitude:        41.8902,
						Longitude:       12.4922,
					},
					{
						Model: gorm.Model{
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						Name:        "Pałac Kultury",
						Country:     "Poland",
						City:        "Warsaw",
						Description: "Built by Soviets",
						ImageURL:    "https://images.unsplash.com/photo-1652821407004-6660b6e4fa9e?q=80&w=1932&auto=format&fit=crop&ixlib=rb-4.0.3&ixid=M3wxMjA3fDB8MHxwaG90by1wYWdlfHx8fGVufDB8fHx8fA%3D%3D",
					},
				}

				return db.CreateInBatches(landmarks, 100).Error
			},
		},
		{
			Name: "AddIndexToCountry",
			Run: func(db *gorm.DB) error {
				return db.Exec("CREATE INDEX IF NOT EXISTS idx_landmarks_country ON landmarks(country)").Error
			},
		},
	}
}
