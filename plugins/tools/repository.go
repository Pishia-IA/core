package tools

import (
	"encoding/json"

	"github.com/Pishia-IA/core/config"
)

var (
	// repository is a repository that contains all the tools.
	repository *ToolRepository
)

type ToolParameter struct {
	Type        string `json:"type"`
	Format      string `json:"format,omitempty"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

type Tools interface {
	// Run is a method that allows the tool to run.
	Run(map[string]interface{}, string) (string, error)
	// Setup sets up the tool, if something is needed before starting the tool.
	Setup() error
	// Description is a method that allows the tool to describe itself.
	Description() string
	// Parameters is a method that allows the tool to describe its parameters.
	Parameters() map[string]*ToolParameter
	// UseCase is a method that allows the tool to describe its use case.
	UseCase() []string
	// NeedConfirmation is a method that allows the tool to know if it needs confirmation.
	NeedConfirmation() bool
}

// ToolRepository is a repository that contains all the tools.
type ToolRepository struct {
	// Tools is a map that contains all the tools.
	Tools map[string]Tools
}

// NewToolRepository creates a new ToolRepository.
func NewToolRepository() *ToolRepository {
	return &ToolRepository{
		Tools: make(map[string]Tools),
	}
}

// DumpToolsJSON dumps the tools to JSON.
func (r *ToolRepository) DumpToolsJSON() (string, error) {
	var tools []map[string]interface{}

	for name, tool := range r.Tools {
		toolMap := map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        name,
				"description": tool.Description(),
				"parameters": map[string]interface{}{
					"type":       "object",
					"properties": make(map[string]map[string]string),
					"required":   []string{},
				},
				"use_case": tool.UseCase(),
			},
		}

		params := tool.Parameters()
		propMap := toolMap["function"].(map[string]interface{})["parameters"].(map[string]interface{})["properties"].(map[string]map[string]string)
		var reqParams []string

		for paramName, param := range params {
			propMap[paramName] = map[string]string{"type": param.Type}
			if param.Required {
				reqParams = append(reqParams, paramName)
			}
		}

		toolMap["function"].(map[string]interface{})["parameters"].(map[string]interface{})["required"] = reqParams
		tools = append(tools, toolMap)
	}

	b, err := json.MarshalIndent(tools, "", "  ")
	if err != nil {
		return "", err
	}

	return string(b), nil
}

// Register registers a tool in the repository.
func (r *ToolRepository) Register(name string, tool Tools) {
	r.Tools[name] = tool
}

// Get gets a tool from the repository.
func (r *ToolRepository) Get(name string) (Tools, bool) {
	tool, ok := r.Tools[name]
	return tool, ok
}

// GetRepository gets the repository.
func GetRepository() *ToolRepository {
	return repository
}

// isToolEnabled checks if the tool is enabled.
func isToolEnabled(tool string, config *config.Base) bool {
	for _, enabledTool := range config.Tool.Enabled {
		if enabledTool == tool {
			return true
		}
	}
	return false
}

// StartTools starts the tools.
func StartTools(config *config.Base) {
	repository = NewToolRepository()

	if isToolEnabled("weather", config) {
		repository.Register("weather", NewWeather(config))
	}
}
