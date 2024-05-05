package config

type Base struct {
	// Assistants is the configuration of the assistants.
	Assistants Assistants `yaml:"assistants"`
	// Tool is the configuration of the tool.
	Tool Tool `yaml:"tool"`
}
