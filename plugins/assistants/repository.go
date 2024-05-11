package assistants

import "github.com/Pishia-IA/core/config"

var (
	// repository is a repository that contains all the assistants.
	repository *AssistantRepository
	// defaultAssistant is the default assistant.
	defaultAssistant string
)

type Assistant interface {
	// SendRequest is a method that allows the assistant to chat with you.
	SendRequest(input string, callback func(output string, err error)) error
	// Setup sets up the assistant, if something is needed before starting the assistant.
	Setup() error
}

// AssistantRepository is a repository that contains all the assistants.
type AssistantRepository struct {
	// Assistants is a map that contains all the assistants.
	Assistants map[string]Assistant
}

// NewAssistantRepository creates a new AssistantRepository.
func NewAssistantRepository() *AssistantRepository {
	return &AssistantRepository{
		Assistants: make(map[string]Assistant),
	}
}

// Register registers an assistant in the repository.
func (r *AssistantRepository) Register(name string, assistant Assistant) {
	r.Assistants[name] = assistant
}

// Get gets an assistant from the repository.
func (r *AssistantRepository) Get(name string) (Assistant, bool) {
	assistant, ok := r.Assistants[name]
	return assistant, ok
}

// GetRepository gets the repository.
func GetRepository() *AssistantRepository {
	return repository
}

// GetDefaultAssistant gets the default assistant.
func GetDefaultAssistant() Assistant {
	assistant, ok := repository.Get(defaultAssistant)
	if !ok {
		return nil
	}
	return assistant
}

// StartAssistants starts the assistants.
func StartAssistants(config *config.Base) {
	repository = NewAssistantRepository()
	repository.Register("ollama", NewOllama(config))
	repository.Register("openai", NewOpenAI(config))

	defaultAssistant = config.Assistants.Plugin
}
