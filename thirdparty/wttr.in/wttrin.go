package wttrin

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type OneWeather struct {
	FeelsLikeC       string `json:"FeelsLikeC"`
	FeelsLikeF       string `json:"FeelsLikeF"`
	Humidity         string `json:"humidity"`
	TempC            string `json:"temp_C"`
	TempF            string `json:"temp_F"`
	LocalObsDateTime string `json:"localObsDateTime"`
	WeatherDesc      []struct {
		Value  string       `json:"value"`
		Hourly []OneWeather `json:"hourly"`
	} `json:"weatherDesc"`
}

type OneWeatherDay struct {
	Astronomy []struct {
		Sunrise string `json:"sunrise"`
		Sunset  string `json:"sunset"`
	} `json:"astronomy"`
	Date   string       `json:"date"`
	Hourly []OneWeather `json:"hourly"`
}

type WeatherInResponse struct {
	CurrentCondition []OneWeather    `json:"current_condition"`
	Weather          []OneWeatherDay `json:"weather"`
}

// FetchWeather fetches the weather of a city.
func FetchWeather(city string) (*WeatherInResponse, error) {
	url := fmt.Sprintf("https://wttr.in/%s?format=j1", city)

	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	var weather WeatherInResponse

	err = json.NewDecoder(response.Body).Decode(&weather)

	if err != nil {
		return nil, err
	}

	return &weather, nil
}
