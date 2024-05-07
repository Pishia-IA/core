package config

// Assistants is the configuration of the assistants.
type Assistants struct {
	// Plugin is the plugin of the assistants.
	Plugin string `yaml:"plugin"`
	// Ollama is the configuration of the Ollama assistant.
	Ollama Ollama `yaml:"ollama,omitempty"`
	// OpenAI is the configuration of the OpenAI assistant.
	OpenAI OpenAI `yaml:"openai,omitempty"`
}

// Ollama is the configuration of the Ollama assistant.
type Ollama struct {
	// Model is the model of the Ollama.
	Model string `yaml:"model"`
	// Endpoint is the endpoint of the Ollama.
	Endpoint string `yaml:"endpoint,omitempty"`
}

// OpenAI is the configuration of the OpenAI assistant.
type OpenAI struct {
	// Model is the model of the OpenAI.
	Model string `yaml:"model"`
	// APIKey is the API key of the OpenAI.
	APIKey string `yaml:"api_key"`
	// Endpoint is the endpoint of the OpenAI.
	Endpoint string `yaml:"endpoint,omitempty"`
}
