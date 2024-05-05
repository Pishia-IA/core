package config

// Assistants is the configuration of the assistants.
type Assistants struct {
	// Plugin is the plugin of the assistants.
	Plugin string `yaml:"plugin"`
	// Ollama is the configuration of the Ollama assistant.
	Ollama Ollama `yaml:"ollama,omitempty"`
}

// Ollama is the configuration of the Ollama assistant.
type Ollama struct {
	// Model is the model of the Ollama.
	Model string `yaml:"model"`
	// Endpoint is the endpoint of the Ollama.
	Endpoint string `yaml:"endpoint,omitempty"`
}
