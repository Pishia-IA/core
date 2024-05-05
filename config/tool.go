package config

// Tool is the configuration of the tool.
type Tool struct {
	// Enabled plugins.
	Enabled []string `yaml:"enabled"`
}
