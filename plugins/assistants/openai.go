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
	toolCall = strings.TrimSpace(toolCall)
	toolCall = strings.ReplaceAll(toolCall, "<tool_call>", "")
	toolCall = strings.ReplaceAll(toolCall, "</tool_call>", "")
	toolCall = strings.TrimSpace(toolCall)

	// If contains ```json, remove it and the last ```
	if strings.Contains(toolCall, "```json") {
		toolCall = strings.ReplaceAll(toolCall, "```json", "")
		toolCall = strings.TrimSpace(toolCall)
		toolCall = strings.TrimSuffix(toolCall, "```")
	}

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
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    "user",
			Content: prompt,
		})
	}

	req := openai.ChatCompletionRequest{
		Model:    o.Model,
		Messages: messages,
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
		Model:    o.Model,
		Messages: o.Chat,
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
		</tool_call>
		`, current_time.Format("2006-01-02"), toolsJSON)),
	})

	return nil
}
