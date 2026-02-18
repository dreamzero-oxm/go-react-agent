package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type CustomLLM struct {
	config *LLMConfig
	client *http.Client
}

func NewCustomLLM(config *LLMConfig) (*CustomLLM, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required for custom LLM")
	}

	return &CustomLLM{
		config: config,
		client: &http.Client{},
	}, nil
}

type CustomRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

type CustomResponse struct {
	Choices []Choice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

type Choice struct {
	Message Message `json:"message"`
}

func (c *CustomLLM) Generate(messages []Message) (string, error) {
	return c.GenerateWithSystem("", messages)
}

func (c *CustomLLM) GenerateWithSystem(systemPrompt string, messages []Message) (string, error) {
	customMessages := make([]Message, 0, len(messages)+1)

	if systemPrompt != "" {
		customMessages = append(customMessages, Message{
			Role:    RoleSystem,
			Content: systemPrompt,
		})
	}

	for _, msg := range messages {
		customMessages = append(customMessages, Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	reqBody := CustomRequest{
		Model:       c.config.Model,
		Messages:    customMessages,
		Temperature: c.config.Temperature,
		MaxTokens:   c.config.MaxTokens,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.config.BaseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := json.Marshal(reqBody)
		return "", fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response CustomResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Error != nil {
		return "", fmt.Errorf("LLM error: %s", response.Error.Message)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return response.Choices[0].Message.Content, nil
}

func (c *CustomLLM) Stream(messages []Message, callback func(chunk string)) error {
	return fmt.Errorf("streaming not implemented for custom LLM")
}

func (c *CustomLLM) Close() error {
	return nil
}
