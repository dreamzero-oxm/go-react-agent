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

type DashScopeLLM struct {
	client *http.Client
	config *LLMConfig
}

type dashscopeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type dashscopeRequest struct {
	Model string              `json:"model"`
	Input struct {
		Messages []dashscopeMessage `json:"messages"`
	} `json:"input"`
	Parameters struct {
		MaxTokens   int     `json:"max_tokens,omitempty"`
		Temperature float64 `json:"temperature,omitempty"`
	} `json:"parameters"`
}

type dashscopeResponse struct {
	Output struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	} `json:"output"`
}

type dashscopeStreamEvent struct {
	Output struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	} `json:"output"`
}

func NewDashScopeLLM(config *LLMConfig) (*DashScopeLLM, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required for DashScope")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation"
	}

	if config.Model == "" {
		config.Model = "qwen-turbo"
	}

	return &DashScopeLLM{
		client: &http.Client{},
		config: config,
	}, nil
}

func (d *DashScopeLLM) Generate(messages []Message) (string, error) {
	return d.GenerateWithSystem("", messages)
}

func (d *DashScopeLLM) GenerateWithSystem(systemPrompt string, messages []Message) (string, error) {
	dashscopeMessages := make([]dashscopeMessage, 0, len(messages)+1)

	if systemPrompt != "" {
		dashscopeMessages = append(dashscopeMessages, dashscopeMessage{
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
		dashscopeMessages = append(dashscopeMessages, dashscopeMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	reqBody := dashscopeRequest{
		Model: d.config.Model,
	}
	reqBody.Input.Messages = dashscopeMessages
	reqBody.Parameters.MaxTokens = d.config.MaxTokens
	reqBody.Parameters.Temperature = d.config.Temperature

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", d.config.BaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.config.APIKey)
	req.Header.Set("X-DashScope-SSE", "disable")

	resp, err := d.client.Do(req)
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

	var dashscopeResp dashscopeResponse
	if err := json.Unmarshal(body, &dashscopeResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(dashscopeResp.Output.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return dashscopeResp.Output.Choices[0].Message.Content, nil
}

func (d *DashScopeLLM) Stream(messages []Message, callback func(chunk string)) error {
	dashscopeMessages := make([]dashscopeMessage, len(messages))
	for i, msg := range messages {
		role := string(msg.Role)
		if role == "assistant" {
			role = "assistant"
		} else {
			role = "user"
		}
		dashscopeMessages[i] = dashscopeMessage{
			Role:    role,
			Content: msg.Content,
		}
	}

	reqBody := dashscopeRequest{
		Model: d.config.Model,
	}
	reqBody.Input.Messages = dashscopeMessages
	reqBody.Parameters.MaxTokens = d.config.MaxTokens
	reqBody.Parameters.Temperature = d.config.Temperature

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", d.config.BaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.config.APIKey)
	req.Header.Set("X-DashScope-SSE", "enable")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := d.client.Do(req)
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
			var event dashscopeStreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			if len(event.Output.Choices) > 0 {
				callback(event.Output.Choices[0].Message.Content)
			}
		}
	}

	return nil
}

func (d *DashScopeLLM) Close() error {
	return nil
}
