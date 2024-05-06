package tools

import (
	"encoding/json"
	"fmt"

	"github.com/Pishia-IA/core/config"
	wttrin "github.com/Pishia-IA/core/thirdparty/wttr.in"
	log "github.com/sirupsen/logrus"
)

type Weather struct {
}

func NewWeather(config *config.Base) *Weather {
	return &Weather{}
}

var (
	cityCalled = make(map[string]bool)
)

func (c *Weather) Run(params map[string]interface{}, userQuery string) (string, error) {
	log.Debugf("Running the Weather tool with the following parameters: %v", params)

	location := params["location"].(string)

	if cityCalled[location] {
		return "I really told you the weather of this location already, please generate a response from your memory.", nil
	}

	cityCalled[location] = true

	weather, err := wttrin.FetchWeather(location)

	if err != nil {
		return "", err
	}

	// Convert weather yo JSON
	weatherJSON, err := json.Marshal(weather)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("[weather-tool] UserQuery: %s, data: %s. NOTE: Some answer could be in a different language that user query, please translate it.", userQuery, weatherJSON), nil
}

func (c *Weather) Setup() error {
	return nil
}

func (c *Weather) NeedConfirmation() bool {
	return false
}

func (c *Weather) Description() string {
	return "Weather is a tool that allows you to get the weather of a location."
}

func (c *Weather) Parameters() map[string]*ToolParameter {
	return map[string]*ToolParameter{
		"location": {
			Type:        "string",
			Format:      "",
			Required:    true,
			Description: "The location that you want to get the weather of.",
		},
		"start_date": {
			Type:        "string",
			Format:      "YYYY-MM-DD",
			Required:    false,
			Description: "The start date of the weather that you want to get.",
		},
		"end_date": {
			Type:        "string",
			Format:      "YYYY-MM-DD",
			Required:    false,
			Description: "The end date of the weather that you want to get.",
		},
	}
}

func (c *Weather) UseCase() []string {
	return []string{
		"Run this tool if the user wants to get the weather of a location.",
		"Run this tool if the user want to know about a weather condition in a location, for example: is raining in New York?",
	}
}
