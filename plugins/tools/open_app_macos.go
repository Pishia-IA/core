package tools

import (
	"encoding/json"
	"os"
	"os/exec"

	"github.com/Pishia-IA/core/config"
	log "github.com/sirupsen/logrus"
)

type OpenAppMacOS struct {
}

func NewOpenAppMacOS(config *config.Base) *OpenAppMacOS {
	return &OpenAppMacOS{}
}

func (c *OpenAppMacOS) Run(params map[string]interface{}, userQuery string) (string, error) {
	log.Debugf("Running the OpenAppMacOS tool with the following parameters: %v", params)
	exec.Command("open", "/Applications/"+params["app"].(string)).Run()

	return "Opening APP...", nil
}

func (c *OpenAppMacOS) Setup() error {
	return nil
}

func (c *OpenAppMacOS) NeedConfirmation() bool {
	return false
}

func (c *OpenAppMacOS) Description() string {
	installedApplications := make([]string, 0)

	entries, err := os.ReadDir("/Applications")

	if err != nil {
		log.Errorf("Error while reading the /Applications folder: %v", err)
	}

	for _, entry := range entries {
		installedApplications = append(installedApplications, entry.Name())
	}

	// Convert to JSON
	installedApplicationsJSON, err := json.Marshal(installedApplications)

	if err != nil {
		log.Errorf("Error while converting the installed applications to JSON: %v", err)
	}

	return "OpenAppMacOS is a tool that allows you to open an application on macOS. The installed applications are: " + string(installedApplicationsJSON)
}

func (c *OpenAppMacOS) Parameters() map[string]*ToolParameter {
	return map[string]*ToolParameter{
		"app": {
			Type:        "string",
			Format:      "",
			Required:    true,
			Description: "The name of the app to open.",
		},
	}
}

func (c *OpenAppMacOS) UseCase() []string {
	return []string{
		"Open an application on macOS.",
	}
}
