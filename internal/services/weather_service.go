package services

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type WeatherData struct {
	Main struct {
		Temp float64 `json:"temp"`
	} `json:"main"`
	Weather []struct {
		Description string `json:"description"`
	} `json:"weather"`
}

func FetchWeatherData(lat, lon float64) (*WeatherData, error) {
	apiKey := "d0e23c5d2a622321138d993e9e7f9f23"
	latStr := strconv.FormatFloat(lat, 'f', 6, 64)
	lonStr := strconv.FormatFloat(lon, 'f', 6, 64)
	url := "http://api.openweathermap.org/data/2.5/weather?lat=" + latStr + "&lon=" + lonStr + "&units=metric&appid=" + apiKey

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var weatherData WeatherData
	if err := json.NewDecoder(resp.Body).Decode(&weatherData); err != nil {
		return nil, err
	}
	return &weatherData, nil
}
