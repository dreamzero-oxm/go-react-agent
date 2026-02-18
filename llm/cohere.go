package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type CohereLLM struct {
	client *http.Client
	config *LLMConfig
}

type cohereMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type cohereRequest struct {
	Model       string           `json:"model"`
	Messages    []cohereMessage  `json:"message,omitempty"`
	ChatHistory []cohereMessage `json:"chat_history,omitempty"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64          `json:"temperature"`
	Stream      bool            `json:"stream"`
}

type cohereResponse struct {
	Text string `json:"text"`
}

type cohereStreamEvent struct {
	EventType string `json:"event_type"`
	Text      string `json:"text"`
}

func NewCohereLLM(config *LLMConfig) (*CohereLLM, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required for Cohere")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://api.cohere.ai/v1/chat"
	}

	if config.Model == "" {
		config.Model = "command-r-plus"
	}

	return &CohereLLM{
		client: &http.Client{},
		config: config,
	}, nil
}

func (c *CohereLLM) Generate(messages []Message) (string, error) {
	return c.GenerateWithSystem("", messages)
}

func (c *CohereLLM) GenerateWithSystem(systemPrompt string, messages []Message) (string, error) {
	var chatHistory []cohereMessage
	var message cohereMessage

	if systemPrompt != "" {
		chatHistory = append(chatHistory, cohereMessage{
			Role:    "SYSTEM",
			Content: systemPrompt,
		})
	}

	if len(messages) > 0 {
		lastMessage := messages[len(messages)-1]
		message = cohereMessage{
			Role:    string(lastMessage.Role),
			Content: lastMessage.Content,
		}

		if len(messages) > 1 {
			for _, msg := range messages[:len(messages)-1] {
				role := string(msg.Role)
				if role == "assistant" {
					role = "CHATBOT"
				} else if role == "user" {
					role = "USER"
				}
				chatHistory = append(chatHistory, cohereMessage{
					Role:    role,
					Content: msg.Content,
				})
			}
		}
	}

	reqBody := cohereRequest{
		Model:       c.config.Model,
		MaxTokens:   c.config.MaxTokens,
		Temperature: c.config.Temperature,
		Stream:      false,
	}

	if message.Content != "" {
		reqBody.Messages = []cohereMessage{message}
	}
	if len(chatHistory) > 0 {
		reqBody.ChatHistory = chatHistory
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.config.BaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.client.Do(req)
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

	var cohereResp cohereResponse
	if err := json.Unmarshal(body, &cohereResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return cohereResp.Text, nil
}

func (c *CohereLLM) Stream(messages []Message, callback func(chunk string)) error {
	var chatHistory []cohereMessage
	var message cohereMessage

	if len(messages) > 0 {
		lastMessage := messages[len(messages)-1]
		message = cohereMessage{
			Role:    string(lastMessage.Role),
			Content: lastMessage.Content,
		}

		if len(messages) > 1 {
			for _, msg := range messages[:len(messages)-1] {
				role := string(msg.Role)
				if role == "assistant" {
					role = "CHATBOT"
				} else if role == "user" {
					role = "USER"
				}
				chatHistory = append(chatHistory, cohereMessage{
					Role:    role,
					Content: msg.Content,
				})
			}
		}
	}

	reqBody := cohereRequest{
		Model:       c.config.Model,
		MaxTokens:   c.config.MaxTokens,
		Temperature: c.config.Temperature,
		Stream:      true,
	}

	if message.Content != "" {
		reqBody.Messages = []cohereMessage{message}
	}
	if len(chatHistory) > 0 {
		reqBody.ChatHistory = chatHistory
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.config.BaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.client.Do(req)
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
		var event cohereStreamEvent
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		if event.EventType == "text-generation" && event.Text != "" {
			callback(event.Text)
		}
	}

	return nil
}

func (c *CohereLLM) Close() error {
	return nil
}
