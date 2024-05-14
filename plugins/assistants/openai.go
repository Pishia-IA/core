package assistants

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Pishia-IA/core/config"
	"github.com/Pishia-IA/core/plugins/tools"
	openai "github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
)

type OpenAI struct {
	// Client is the client of the OpenAI.
	Client *openai.Client
	// Chat is the chat of the OpenAI.
	Chat []openai.ChatCompletionMessage
	// Model is the model of the OpenAI.
	Model string `yaml:"model"`
}

// NewOpenAI creates a new OpenAI.
func NewOpenAI(config *config.Base) *OpenAI {
	openaiConfig := openai.DefaultConfig(config.Assistants.OpenAI.APIKey)
	openaiConfig.BaseURL = config.Assistants.OpenAI.Endpoint
	return &OpenAI{
		Client: openai.NewClientWithConfig(openaiConfig),
		Chat:   make([]openai.ChatCompletionMessage, 0),
		Model:  config.Assistants.OpenAI.Model,
	}
}

func (o *OpenAI) processToolCall(toolCall string) (string, error) {
	// Get only the content between the <tool_call> tags, other text can be ignored
	toolCall = strings.Split(toolCall, "<tool_call>")[1]
	toolCall = strings.Split(toolCall, "</tool_call>")[0]
	toolCall = strings.TrimSpace(toolCall)

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
		processedPrompts := make([]string, len(toolResponse.Prompts))
		errChan := make(chan error, len(toolResponse.Prompts))

		for i, prompt := range toolResponse.Prompts {
			go func(i int, prompt string) {
				result, err := o.SendRequestWithnoMemory([]string{fmt.Sprintf("Please summarize and extract the key information from the following text: %s", prompt)})
				if err != nil {
					errChan <- err
					return
				}
				processedPrompts[i] = result
				errChan <- nil
			}(i, prompt)
		}

		for range toolResponse.Prompts {
			err := <-errChan
			if err != nil {
				log.Warnf("Error processing prompt: %s", err.Error())
			}
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

// SendRequestWithnoMemoryAndModel sends a request to the OpenAI without memory and with a specific model.
func (o *OpenAI) SendRequestWithnoMemoryAndModel(prompts []string, model string) (string, error) {
	messages := make([]openai.ChatCompletionMessage, 0)

	for _, prompt := range prompts {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    "user",
			Content: prompt,
		})
	}

	req := openai.ChatCompletionRequest{
		Model:    model,
		Messages: messages,
	}

	resp, err := o.Client.CreateChatCompletion(context.Background(), req)

	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}

// SendRequestWithnoMemory sends a request to the OpenAI without memory.
func (o *OpenAI) SendRequestWithnoMemory(prompts []string) (string, error) {
	messages := make([]openai.ChatCompletionMessage, 0)

	for _, prompt := range prompts {
		if len(prompt) == 0 {
			continue
		}
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    "user",
			Content: prompt,
		})
	}

	req := openai.ChatCompletionRequest{
		Model:       o.Model,
		Messages:    messages,
		Temperature: 0,
	}

	resp, err := o.Client.CreateChatCompletion(context.Background(), req)

	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}

// SendRequest sends a request to the OpenAI.
func (o *OpenAI) SendRequest(prompt string, callback func(output string, err error)) error {
	o.Chat = append(o.Chat, openai.ChatCompletionMessage{
		Role:    "user",
		Content: prompt,
	})

	req := openai.ChatCompletionRequest{
		Model:       o.Model,
		Messages:    o.Chat,
		Temperature: 0,
	}

	stream, err := o.Client.CreateChatCompletionStream(context.Background(), req)

	if err != nil {
		callback("", err)
		return nil
	}

	defer stream.Close()

	fullContent := ""
	toolMode := false

	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			callback("", err)
			return nil
		}

		if len(response.Choices) > 0 && len(response.Choices[0].Delta.Content) > 0 && response.Choices[0].Delta.Content[0] == '<' {
			toolMode = true
		}

		fullContent += response.Choices[0].Delta.Content

		if toolMode {
			continue
		}

		callback(response.Choices[0].Delta.Content, nil)
	}

	o.Chat = append(o.Chat, openai.ChatCompletionMessage{
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

		o.Chat = append(o.Chat, openai.ChatCompletionMessage{
			Role:    "assistant",
			Content: toolCall,
		})

		callback(toolCall+"\n", nil)
		return nil
	}

	return nil
}

// Setup sets up the OpenAI assistant.
func (o *OpenAI) Setup() error {
	current_time := time.Now().Local()

	toolsJSON, err := tools.GetRepository().DumpToolsJSON()

	if err != nil {
		return err
	}

	o.Chat = append(o.Chat, openai.ChatCompletionMessage{
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
