package config

import (
	"os"
	"path/filepath"

	"github.com/kirsle/configdir"

	"gopkg.in/yaml.v3"

	log "github.com/sirupsen/logrus"
)

var (
	// configDir is the directory where the configuration files are stored.
	configDir string
)

// GetCurrentConfigurations gets the current configurations.
func GetCurrentConfigurations() (*Base, error) {
	var base Base
	if !DoesConfigExist() {
		log.Warn("Configuration does not exist.")
		log.Debug("Creating a new configuration.")

		err := CreateDefaultConfig()

		if err != nil {
			return nil, err
		}
	}
	err := LoadConfig(&base)
	if err != nil {
		return nil, err
	}
	return &base, nil
}

// DoesConfigExist checks if the configuration exists.
func DoesConfigExist() bool {
	configPath := filepath.Join(configDir, "config.yaml")
	_, err := os.Stat(configPath)
	return !os.IsNotExist(err)
}

// CreateDefaultConfig creates the default configuration.
func CreateDefaultConfig() error {
	defaultConfiguration := Base{
		Assistants: Assistants{
			Plugin: "ollama",
			Ollama: Ollama{
				Model:    "adrienbrault/nous-hermes2pro:Q8_0",
				Endpoint: "http://localhost:11434",
			},
			OpenAI: OpenAI{
				Model:    "gpt-4o",
				APIKey:   "<api_key>",
				Endpoint: "https://api.openai.com/v1/",
			},
		},
		Tool: Tool{},
	}

	// Create the configuration directory
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		return err
	}

	// Create the configuration file
	configPath := filepath.Join(configDir, "config.yaml")
	yamlFile, err := yaml.Marshal(defaultConfiguration)
	if err != nil {
		return err
	}
	err = os.WriteFile(configPath, yamlFile, 0644)
	if err != nil {
		return err
	}

	return nil
}

// LoadConfig loads the configuration.
func LoadConfig(config interface{}) error {
	configPath := filepath.Join(configDir, "config.yaml")

	log.Debug("Loading configuration from ", configPath)

	// Read configPath
	yamlFile, err := os.ReadFile(configPath)

	if err != nil {
		return err
	}

	// Unmarshal yamlFile
	err = yaml.Unmarshal(yamlFile, config)
	if err != nil {
		return err
	}

	return nil
}

func init() {
	configDir = configdir.LocalConfig("pishia")
}
