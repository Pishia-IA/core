package tools

import (
	"github.com/Pishia-IA/core/config"
	log "github.com/sirupsen/logrus"
)

type Weather struct {
}

func NewWeather(config *config.Base) *Weather {
	return &Weather{}
}

func (c *Weather) Run(params map[string]interface{}) (string, error) {
	log.Debugf("Running the Weather tool with the following parameters: %v", params)
	return "27º Sunny day", nil
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
	}
}
