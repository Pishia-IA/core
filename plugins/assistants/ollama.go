package assistants

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Pishia-IA/core/config"
	"github.com/Pishia-IA/core/plugins/tools"
	"github.com/Pishia-IA/core/thirdparty/ollama"
	log "github.com/sirupsen/logrus"
)

// Ollama is an assistant that can chat with you.
type Ollama struct {
	// Endpoint is the endpoint of the Ollama.
	Client *ollama.OllamaClient
	// Chat is the chat of the Ollama.
	Chat []ollama.Message
	// Model is the model of the Ollama.
	Model string `yaml:"model"`
}

// NewDefaultOllama creates a new Ollama.
func NewDefaultOllama() *Ollama {
	return &Ollama{
		Client: ollama.NewOllamaClient("http://localhost:11434"),
		Chat:   []ollama.Message{},
		Model:  "adrienbrault/nous-hermes2pro:Q8_0",
	}
}

// NewOllama creates a new Ollama.
func NewOllama(config *config.Base) *Ollama {
	return &Ollama{
		Client: ollama.NewOllamaClient(config.Assistants.Ollama.Endpoint),
		Chat:   []ollama.Message{},
		Model:  config.Assistants.Ollama.Model,
	}
}

// processToolCall processes the tool call.
func (o *Ollama) processToolCall(toolCall string) (string, error) {
	log.Debugf("Processing tool call: %s", toolCall)
	toolCall = strings.TrimSpace(toolCall)
	toolCall = strings.TrimPrefix(toolCall, "<tool_call>")
	toolCall = strings.TrimSuffix(toolCall, "</tool_call>")
	toolCall = strings.TrimSpace(toolCall)

	var toolCallJSON map[string]interface{}

	err := json.Unmarshal([]byte(toolCall), &toolCallJSON)

	if err != nil {
		return "", err
	}

	toolName, ok := toolCallJSON["name"].(string)

	if !ok {
		return "", fmt.Errorf("tool name is not a string")
	}

	toolArguments, ok := toolCallJSON["arguments"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("tool arguments is not a map")
	}

	tool, ok := tools.GetRepository().Get(toolName)

	if !ok {
		return "", fmt.Errorf("tool %s not found", toolName)
	}

	userQuery := o.Chat[len(o.Chat)-2].Content
	toolResponse, err := tool.Run(toolArguments, userQuery)

	if err != nil {
		return "", err
	}

	return toolResponse, nil
}

// SendRequest is a method that allows the Ollama to chat with you.
func (o *Ollama) SendRequest(input string) (string, error) {

	o.Chat = append(o.Chat, ollama.Message{
		Role:    "user",
		Content: input,
	})

	resp, err := o.Client.Chat(&ollama.ChatRequest{
		Model:    o.Model,
		Messages: o.Chat,
	})

	if err != nil {
		return "", err
	}

	o.Chat = append(o.Chat, resp.Message)

	if strings.Contains(resp.Message.Content, "<tool_call>") {
		log.Debug("Tool call detected")
		toolCall, err := o.processToolCall(resp.Message.Content)

		if err != nil {
			o.Chat = o.Chat[:len(o.Chat)-1]
			return "", err
		}

		o.Chat = append(o.Chat, ollama.Message{
			Role:    "system",
			Content: toolCall,
		})

		resp, err := o.Client.Chat(&ollama.ChatRequest{
			Model:    o.Model,
			Messages: o.Chat,
		})

		if err != nil {
			o.Chat = o.Chat[:len(o.Chat)-1]
			return "", err
		}

		o.Chat = append(o.Chat, resp.Message)

		return resp.Message.Content, nil

	}

	return resp.Message.Content, nil
}

// Setup sets up the Ollama, if something is needed before starting the Ollama.
func (o *Ollama) Setup() error {
	_, err := o.Client.ShowModel(&ollama.ShowModelRequest{
		Name: o.Model,
	})

	if err != nil {
		log.Debugf("Model not found: %v", err)
		log.Debugf("Pulling model: %v", o.Model)

		_, err := o.Client.PullModel(&ollama.PullModelRequest{
			Name:   o.Model,
			Stream: false,
		})

		if err != nil {
			return err
		}
	}

	current_time := time.Now().Local()

	toolsJSON, err := tools.GetRepository().DumpToolsJSON()

	if err != nil {
		return err
	}

	o.Chat = append(o.Chat, ollama.Message{
		Role: "system",
		Content: strings.Trim(fmt.Sprintf(`
		Current time: %s
		Your name is PishIA.
		You are a function calling AI model. You are provided with function signatures within <tools></tools> XML tags. You may call one or more functions to assist with the user query. Don't make assumptions about what values to plug into functions. Here are the available tools:
		<tools>
		%s
		</tools>
		Instructions:
		- You only have to use a function, if use_case match with user query.
		- Only tools defined in <tools></tools> XML tags are available for use, you musn't use any other tool.
		- Use the following pydantic model json schema for each tool call you will make: {"properties": {"arguments": {"title": "Arguments", "type": "object"}, "name": {"title": "Name", "type": "string"}}, "required": ["arguments", "name"], "title": "FunctionCall", "type": "object"} For each function call return a json object with function name and arguments within <tool_call></tool_call> XML tags as follows:
		<tool_call>
		{"arguments": <args-dict>, "name": <function-name>}
		</tool_call><|im_end|>
`, current_time.Format("2006-01-02"), toolsJSON), "\n"),
	})

	o.Chat = append(o.Chat, ollama.Message{
		Role:    "user",
		Content: "Hello",
	})

	return nil
}
