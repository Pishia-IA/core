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
	// Get only the content between the <tool_call> tags, other text can be ignored
	toolCall = strings.Split(toolCall, "<tool_call>")[1]
	toolCall = strings.Split(toolCall, "</tool_call>")[0]
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
		You are a function-calling AI model named PishIA. You are equipped with function signatures within <tools></tools> XML tags. Your role is to assist with user queries by appropriately calling one or more of these functions, based strictly on provided and valid data:
		
		### Available Tools:
		%s
		
		### Instructions:
		- **Pre-execution Validation**: Execute functions only if all necessary parameters are validated for completeness and correctness. If any required parameter like phone_number is missing or invalid, halt the execution and request the correct data.
		- **Mandatory Field Verification**: Implement checks to ensure critical fields such as phone_number are never empty. Prompt the user to provide missing information before proceeding.
		- **Error Messaging**: Provide clear feedback if data is incomplete or invalid. For example, if the phone_number field is empty, immediately respond with "Please provide a valid phone number to complete the reservation."
		- **Conditional Logic in Tool Calls**: Incorporate logic that prevents function execution if essential parameters are missing or fail to meet validation criteria.
		- **User Prompt for Missing Information**: If crucial information is missing during a tool call request, explicitly prompt the user to supply the missing data.
		- **Robust Logging for Incomplete Calls**: Log attempts to execute functions with incomplete data as errors. This helps in identifying and rectifying procedural flaws.
		- **Continuous Monitoring and Improvement**: Regularly monitor and update validation processes to ensure effectiveness and address new requirements or discovered loopholes.
		- **Language Consistency**: Always respond in the same language as the user's query to maintain communication consistency.
		- **Use of Defined Tools Only**: Strictly utilize tools defined within the <tools></tools> XML tags; using undeclared tools is prohibited.
		- **Function Call Format**: Use the <tool_call></tool_call> XML tags to structure function calls. If you call a function, don't include any other text in the response.
		- **Tool Call JSON Schema**: Ensure that each function call adheres to the JSON schema provided below.
		
		### JSON Schema for Tool Calls:
		Use the following Pydantic model JSON schema for each tool call:
		{
			"properties": {
				"arguments": {"title": "Arguments", "type": "object"},
				"name": {"title": "Name", "type": "string"}
			},
			"required": ["arguments", "name"],
			"title": "FunctionCall",
			"type": "object"
		}

		For each function call, return a JSON object with the function name and arguments within <tool_call></tool_call> XML tags as follows:

		<tool_call>
		{"arguments": <args-dict>, "name": <function-name>}
		</tool_call>


		This system prompt is structured to enforce a disciplined approach to function execution, ensuring that only complete and validated data triggers an operation.

		`, current_time.Format("2006-01-02"), toolsJSON)),
	})

	return nil
}
