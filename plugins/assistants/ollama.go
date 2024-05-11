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
	toolCall = strings.TrimSpace(toolCall)
	toolCall = strings.ReplaceAll(toolCall, "<tool_call>", "")
	toolCall = strings.ReplaceAll(toolCall, "</tool_call>", "")
	toolCall = strings.TrimSpace(toolCall)

	// Replace ' by " to avoid json unmarshal error
	toolCall = strings.ReplaceAll(toolCall, "'", "\"")

	log.Debugf("Processing tool call: %s", toolCall)

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
	// Check if origin_query is present in the arguments
	searchQuery, ok := toolArguments["search"].(string)

	if ok {
		userQuery = searchQuery
	}

	toolResponse, err := tool.Run(toolArguments, userQuery)

	if err != nil {
		return "", err
	}

	switch toolResponse.Type {
	case "string":
		return toolResponse.Data, nil
	case "prompt":
		processedPrompts := make([]string, 0)

		for _, prompt := range toolResponse.Prompts {
			result, err := o.SendRequestWithnoMemory([]string{fmt.Sprintf("Please summarize and extract the key information from the following text: %s", prompt)})
			if err != nil {
				log.Warnf("Error processing prompt: %s", err.Error())
				continue
			}

			processedPrompts = append(processedPrompts, result)
		}

		processedPrompts = append(processedPrompts, fmt.Sprintf("user query: %s\n NOTE: Be concise, short and specific, and you must answer with the same language as the user query.", userQuery))

		log.Debugf("Tool prompts: %v", processedPrompts)
		result, err := o.SendRequestWithnoMemory(processedPrompts)

		if err != nil {
			return "", err
		}

		return result, nil
	}

	return "", fmt.Errorf("unknown tool response type")
}

// SendRequestWithnoMemory is a method that allows the Ollama to chat with you without memory.
func (o *Ollama) SendRequestWithnoMemory(input []string) (string, error) {
	messages := []ollama.Message{}

	inputsWithoutLast := input[:len(input)-1]
	lastInput := input[len(input)-1]

	if len(inputsWithoutLast) > 0 {
		messages = append(messages, ollama.Message{
			Role:    "user",
			Content: fmt.Sprintf("Here some sources to help you: %s", strings.Join(inputsWithoutLast, "\n")),
		})
	}

	messages = append(messages, ollama.Message{
		Role:    "user",
		Content: lastInput,
	})

	resp, err := o.Client.Chat(&ollama.ChatRequest{
		Model:    o.Model,
		Messages: messages,
	})

	if err != nil {
		return "", err
	}

	log.Debugf("Response: %s", resp.Message.Content)
	return resp.Message.Content, nil
}

// SendRequestWithNoMemoryCustomModel is a method that allows the Ollama to chat with you without memory and with a custom model.
func (o *Ollama) SendRequestWithNoMemoryCustomModel(input string, model string) (string, error) {
	resp, err := o.Client.Chat(&ollama.ChatRequest{
		Model: model,
		Messages: []ollama.Message{
			{
				Role:    "user",
				Content: input,
			},
		},
	})

	if err != nil {
		return "", err
	}

	return resp.Message.Content, nil
}

// SendRequest is a method that allows the Ollama to chat with you.
func (o *Ollama) SendRequest(input string, callback func(output string, err error)) error {
	if callback == nil {
		return fmt.Errorf("callback is nil")
	}

	o.Chat = append(o.Chat, ollama.Message{
		Role:    "user",
		Content: input,
	})

	chanResp, chanErr, err := o.Client.ChatStream(&ollama.ChatRequest{
		Model:    o.Model,
		Messages: o.Chat,
	})

	if err != nil {
		callback("", err)
		return nil
	}

	toolMode := false
	fullContent := ""
	inProgress := true

	for inProgress {
		select {
		case resp := <-chanResp:
			if resp.Done {
				callback(resp.Message.Content+"\n", nil)
				inProgress = false
				continue
			}

			fullContent += resp.Message.Content

			if len(resp.Message.Content) > 0 && resp.Message.Content[0] == '<' {
				toolMode = true
			}

			if toolMode {
				continue
			}

			callback(resp.Message.Content, nil)

		case err := <-chanErr:
			callback("", err)
			return nil
		}
	}

	o.Chat = append(o.Chat, ollama.Message{
		Role:    "assistant",
		Content: fullContent,
	})

	if toolMode && strings.Contains(fullContent, "<tool_call>") {
		log.Debug("Tool call detected")
		toolCall, err := o.processToolCall(fullContent)

		if err != nil {
			o.Chat = o.Chat[:len(o.Chat)-1]
			callback("", err)
			return nil
		}

		o.Chat = append(o.Chat, ollama.Message{
			Role:    "assistant",
			Content: toolCall,
		})

		callback(toolCall+"\n", nil)
		return nil
	}

	return nil
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
		Content: strings.TrimSpace(fmt.Sprintf(`Today date: %s
Knowledge cutoff: 2023-12-31
You are a function calling AI model, your name is PishIA. You are provided with function signatures within <tools></tools> XML tags. You may call one or more functions to assist with the user query. Don't make assumptions about what values to plug into functions. Here are the available tools:
<tools>
%s
</tools>
Instructions:
- If you use a function, you must only have to answer with the tool call,no extra information.
- In case of not using a function, you must answer with your knowledge.
- Be sure to include all required parameters for the function.
- You only have to use a function, if use_case match with user query.
- If you need more information for running a tool, ask the user for missing parameters.
- If the user ask something using a relative date, use today date as reference.
- Only tools defined in <tools></tools> XML tags are available for use, you musn't use any other tool.
- You must answer with the same language as the user query.
- Use the following pydantic model json schema for each tool call you will make: {"properties": {"arguments": {"title": "Arguments", "type": "object"}, "name": {"title": "Name", "type": "string"}}, "required": ["arguments", "name"], "title": "FunctionCall", "type": "object"} For each function call return a json object with function name and arguments within <tool_call></tool_call> XML tags as follows:
<tool_call>
{"arguments": <args-dict>, "name": <function-name>}
</tool_call><|im_end|>
`, current_time.Format("2006-01-02"), toolsJSON)),
	})

	return nil
}
