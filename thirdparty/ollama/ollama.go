package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// OllamaClient is a client for the Ollama.
type OllamaClient struct {
	// Endpoint is the endpoint of the Ollama.
	Endpoint string
	// HTTPClient is the HTTP client of the Ollama.
	HTTPClient *http.Client
}

// NewOllamaClient creates a new OllamaClient.
func NewOllamaClient(endpoint string) *OllamaClient {
	return &OllamaClient{
		Endpoint:   endpoint,
		HTTPClient: &http.Client{},
	}
}

// ShowModelRequest is a request to show a model.
type ShowModelRequest struct {
	// Name is the name of the model.
	Name string `json:"name"`
}

// ShowModelResponse is a response to show a model.
type ShowModelResponse struct {
	// Model is the model.
	ModelFile string `json:"modelfile"`

	// Parameteres
	Parameters string `json:"parameters"`

	// Template	is the template
	Template string `json:"template"`
}

// ShowModel shows a model.
func (c *OllamaClient) ShowModel(req *ShowModelRequest) (*ShowModelResponse, error) {
	reqJSON, err := json.Marshal(req)

	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Post(c.Endpoint+"/api/show", "application/json", bytes.NewBuffer(reqJSON))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var showModelResp ShowModelResponse

	err = json.NewDecoder(resp.Body).Decode(&showModelResp)

	if err != nil {
		return nil, err
	}

	return &showModelResp, nil
}

// PullModelRequest is a request to pull a model.
type PullModelRequest struct {
	// Name is the name of the model.
	Name string `json:"name"`
	// Stream is the stream of the model.
	Stream bool `json:"stream"`
}

// PullModelResponse is a response to pull a model.
type PullModelResponse struct {
	// Status is the status of the model.
	Status string `json:"status"`
}

// PullModel pulls a model.
func (c *OllamaClient) PullModel(req *PullModelRequest) (*PullModelResponse, error) {
	reqJSON, err := json.Marshal(req)

	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Post(c.Endpoint+"/api/pull", "application/json", bytes.NewBuffer(reqJSON))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var pullModelResp PullModelResponse

	err = json.NewDecoder(resp.Body).Decode(&pullModelResp)

	if err != nil {
		return nil, err
	}

	return &pullModelResp, nil
}

// Message is a message that can be sent to the Ollama.
type Message struct {
	// Role is the role of the message.
	Role string `json:"role"`
	// Content is the content of the message.
	Content string `json:"content"`
}

// ChatRequest is a request to chat with the Ollama.
type ChatRequest struct {
	// Model is the model of the Ollama.
	Model string `json:"model"`
	// Messages is the messages of the Ollama.
	Messages []Message `json:"messages"`
	// Stream is the stream of the model.
	Stream bool `json:"stream"`
}

// ChatResponse is a response to chat with the Ollama.
type ChatResponse struct {
	// Model is the model of the Ollama.
	Model string `json:"model"`
	// Mesage is the message of the Ollama.
	Message Message `json:"message"`
	// Done is the done of the Ollama.
	Done bool `json:"done"`
}

// Chat chats with the Ollama.
func (c *OllamaClient) Chat(req *ChatRequest) (*ChatResponse, error) {
	reqJSON, err := json.Marshal(req)

	if err != nil {
		return nil, err
	}

	req.Stream = false // We don't support streaming yet.

	resp, err := c.HTTPClient.Post(c.Endpoint+"/api/chat", "application/json", bytes.NewBuffer(reqJSON))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var chatResp ChatResponse

	err = json.NewDecoder(resp.Body).Decode(&chatResp)

	if err != nil {
		return nil, err
	}

	return &chatResp, nil
}
