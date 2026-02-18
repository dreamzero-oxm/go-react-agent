package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type AnthropicLLM struct {
	client *http.Client
	config *LLMConfig
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	Messages  []anthropicMessage `json:"messages"`
	MaxTokens int                `json:"max_tokens"`
	Stream    bool               `json:"stream"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

type anthropicStreamEvent struct {
	Type string `json:"type"`
	Delta struct {
		Text string `json:"text"`
	} `json:"delta"`
}

func NewAnthropicLLM(config *LLMConfig) (*AnthropicLLM, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required for Anthropic")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com/v1/messages"
	}

	if config.Model == "" {
		config.Model = "claude-3-sonnet-20240229"
	}

	return &AnthropicLLM{
		client: &http.Client{},
		config: config,
	}, nil
}

func (a *AnthropicLLM) Generate(messages []Message) (string, error) {
	return a.GenerateWithSystem("", messages)
}

func (a *AnthropicLLM) GenerateWithSystem(systemPrompt string, messages []Message) (string, error) {
	anthropicMessages := make([]anthropicMessage, 0, len(messages))

	if systemPrompt != "" {
		anthropicMessages = append(anthropicMessages, anthropicMessage{
			Role:    "user",
			Content: systemPrompt,
		})
	}

	for _, msg := range messages {
		anthropicMessages = append(anthropicMessages, anthropicMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	reqBody := anthropicRequest{
		Model:     a.config.Model,
		Messages:  anthropicMessages,
		MaxTokens: a.config.MaxTokens,
		Stream:    false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", a.config.BaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: %s", string(body))
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return anthropicResp.Content[0].Text, nil
}

func (a *AnthropicLLM) Stream(messages []Message, callback func(chunk string)) error {
	anthropicMessages := make([]anthropicMessage, len(messages))
	for i, msg := range messages {
		anthropicMessages[i] = anthropicMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	reqBody := anthropicRequest{
		Model:     a.config.Model,
		Messages:  anthropicMessages,
		MaxTokens: a.config.MaxTokens,
		Stream:    true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", a.config.BaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", string(body))
	}

	decoder := json.NewDecoder(resp.Body)
	for {
		var event anthropicStreamEvent
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		if event.Type == "content_block_delta" {
			callback(event.Delta.Text)
		}
	}

	return nil
}

func (a *AnthropicLLM) Close() error {
	return nil
}
