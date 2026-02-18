package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type GenericLLM struct {
	client *http.Client
	config *LLMConfig
}

type genericRequest struct {
	Messages    []genericMessage `json:"messages"`
	Model       string           `json:"model,omitempty"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
	Stream      bool             `json:"stream,omitempty"`
}

type genericMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type genericResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type genericStreamEvent struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func NewGenericLLM(config *LLMConfig) (*GenericLLM, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("BaseURL is required for Generic LLM")
	}

	if config.Model == "" {
		config.Model = "default-model"
	}

	return &GenericLLM{
		client: &http.Client{},
		config: config,
	}, nil
}

func (g *GenericLLM) Generate(messages []Message) (string, error) {
	return g.GenerateWithSystem("", messages)
}

func (g *GenericLLM) GenerateWithSystem(systemPrompt string, messages []Message) (string, error) {
	genericMessages := make([]genericMessage, 0, len(messages)+1)

	if systemPrompt != "" {
		genericMessages = append(genericMessages, genericMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	for _, msg := range messages {
		genericMessages = append(genericMessages, genericMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	reqBody := genericRequest{
		Model:       g.config.Model,
		Messages:    genericMessages,
		MaxTokens:   g.config.MaxTokens,
		Temperature: g.config.Temperature,
		Stream:      false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", g.config.BaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if g.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+g.config.APIKey)
	}

	resp, err := g.client.Do(req)
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

	var genericResp genericResponse
	if err := json.Unmarshal(body, &genericResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(genericResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return genericResp.Choices[0].Message.Content, nil
}

func (g *GenericLLM) Stream(messages []Message, callback func(chunk string)) error {
	genericMessages := make([]genericMessage, len(messages))
	for i, msg := range messages {
		genericMessages[i] = genericMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	reqBody := genericRequest{
		Model:       g.config.Model,
		Messages:    genericMessages,
		MaxTokens:   g.config.MaxTokens,
		Temperature: g.config.Temperature,
		Stream:      true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", g.config.BaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if g.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+g.config.APIKey)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", string(body))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			if data == "[DONE]" {
				break
			}

			var event genericStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			if len(event.Choices) > 0 && event.Choices[0].Delta.Content != "" {
				callback(event.Choices[0].Delta.Content)
			}
		}
	}

	return nil
}

func (g *GenericLLM) Close() error {
	return nil
}
