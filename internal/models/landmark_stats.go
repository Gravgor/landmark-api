package models

type LandmarkStats struct {
	TotalLandmarks      int64            `json:"totalLandmarks"`
	LandmarksByCategory map[string]int64 `json:"landmarksByCategory"`
	LandmarksByCountry  map[string]int64 `json:"landmarksByCountry"`
	RecentlyAdded       []Landmark       `json:"recentlyAdded"`
}
