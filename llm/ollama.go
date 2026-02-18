package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type OllamaLLM struct {
	client *http.Client
	config *LLMConfig
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaRequest struct {
	Model    string           `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool             `json:"stream"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

type ollamaResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

type ollamaStreamEvent struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

func NewOllamaLLM(config *LLMConfig) (*OllamaLLM, error) {
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:11434/api/chat"
	}

	if config.Model == "" {
		config.Model = "llama2"
	}

	return &OllamaLLM{
		client: &http.Client{},
		config: config,
	}, nil
}

func (o *OllamaLLM) Generate(messages []Message) (string, error) {
	return o.GenerateWithSystem("", messages)
}

func (o *OllamaLLM) GenerateWithSystem(systemPrompt string, messages []Message) (string, error) {
	ollamaMessages := make([]ollamaMessage, 0, len(messages)+1)

	if systemPrompt != "" {
		ollamaMessages = append(ollamaMessages, ollamaMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	for _, msg := range messages {
		role := string(msg.Role)
		if role == "assistant" {
			role = "assistant"
		} else {
			role = "user"
		}
		ollamaMessages = append(ollamaMessages, ollamaMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	reqBody := ollamaRequest{
		Model:    o.config.Model,
		Messages: ollamaMessages,
		Stream:   false,
	}

	if o.config.Temperature > 0 || o.config.MaxTokens > 0 {
		reqBody.Options = make(map[string]interface{})
		if o.config.Temperature > 0 {
			reqBody.Options["temperature"] = o.config.Temperature
		}
		if o.config.MaxTokens > 0 {
			reqBody.Options["num_predict"] = o.config.MaxTokens
		}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", o.config.BaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
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

	var ollamaResp ollamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return ollamaResp.Message.Content, nil
}

func (o *OllamaLLM) Stream(messages []Message, callback func(chunk string)) error {
	ollamaMessages := make([]ollamaMessage, len(messages))
	for i, msg := range messages {
		role := string(msg.Role)
		if role == "assistant" {
			role = "assistant"
		} else {
			role = "user"
		}
		ollamaMessages[i] = ollamaMessage{
			Role:    role,
			Content: msg.Content,
		}
	}

	reqBody := ollamaRequest{
		Model:    o.config.Model,
		Messages: ollamaMessages,
		Stream:   true,
	}

	if o.config.Temperature > 0 || o.config.MaxTokens > 0 {
		reqBody.Options = make(map[string]interface{})
		if o.config.Temperature > 0 {
			reqBody.Options["temperature"] = o.config.Temperature
		}
		if o.config.MaxTokens > 0 {
			reqBody.Options["num_predict"] = o.config.MaxTokens
		}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", o.config.BaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
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

		var event ollamaStreamEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		if event.Message.Content != "" {
			callback(event.Message.Content)
		}

		if event.Done {
			break
		}
	}

	return nil
}

func (o *OllamaLLM) Close() error {
	return nil
}
