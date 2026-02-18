package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type MistralLLM struct {
	client *http.Client
	config *LLMConfig
}

type mistralMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type mistralRequest struct {
	Model       string           `json:"model"`
	Messages    []mistralMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64          `json:"temperature"`
	Stream      bool            `json:"stream"`
}

type mistralResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type mistralStreamEvent struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func NewMistralLLM(config *LLMConfig) (*MistralLLM, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required for Mistral")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.mistral.ai/v1/chat/completions"
	}

	if config.Model == "" {
		config.Model = "mistral-large-latest"
	}

	return &MistralLLM{
		client: &http.Client{},
		config: config,
	}, nil
}

func (m *MistralLLM) Generate(messages []Message) (string, error) {
	return m.GenerateWithSystem("", messages)
}

func (m *MistralLLM) GenerateWithSystem(systemPrompt string, messages []Message) (string, error) {
	mistralMessages := make([]mistralMessage, 0, len(messages)+1)

	if systemPrompt != "" {
		mistralMessages = append(mistralMessages, mistralMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	for _, msg := range messages {
		mistralMessages = append(mistralMessages, mistralMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	reqBody := mistralRequest{
		Model:       m.config.Model,
		Messages:    mistralMessages,
		MaxTokens:   m.config.MaxTokens,
		Temperature: m.config.Temperature,
		Stream:      false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", m.config.BaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.config.APIKey)

	resp, err := m.client.Do(req)
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

	var mistralResp mistralResponse
	if err := json.Unmarshal(body, &mistralResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(mistralResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return mistralResp.Choices[0].Message.Content, nil
}

func (m *MistralLLM) Stream(messages []Message, callback func(chunk string)) error {
	mistralMessages := make([]mistralMessage, len(messages))
	for i, msg := range messages {
		mistralMessages[i] = mistralMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	reqBody := mistralRequest{
		Model:       m.config.Model,
		Messages:    mistralMessages,
		MaxTokens:   m.config.MaxTokens,
		Temperature: m.config.Temperature,
		Stream:      true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", m.config.BaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.config.APIKey)

	resp, err := m.client.Do(req)
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
		var event mistralStreamEvent
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		if len(event.Choices) > 0 && event.Choices[0].Delta.Content != "" {
			callback(event.Choices[0].Delta.Content)
		}
	}

	return nil
}

func (m *MistralLLM) Close() error {
	return nil
}
