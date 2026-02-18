package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type GeminiLLM struct {
	client *http.Client
	config *LLMConfig
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
	Role string       `json:"role,omitempty"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiRequest struct {
	Contents    []geminiContent `json:"contents"`
	GenerationConfig struct {
		MaxOutputTokens int `json:"maxOutputTokens"`
		Temperature     float64 `json:"temperature"`
	} `json:"generationConfig"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
}

type geminiStreamEvent struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
}

func NewGeminiLLM(config *LLMConfig) (*GeminiLLM, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required for Gemini")
	}

	if config.BaseURL == "" {
		config.BaseURL = fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", config.Model, config.APIKey)
	}

	if config.Model == "" {
		config.Model = "gemini-pro"
	}

	return &GeminiLLM{
		client: &http.Client{},
		config: config,
	}, nil
}

func (g *GeminiLLM) Generate(messages []Message) (string, error) {
	return g.GenerateWithSystem("", messages)
}

func (g *GeminiLLM) GenerateWithSystem(systemPrompt string, messages []Message) (string, error) {
	contents := make([]geminiContent, 0, len(messages)+1)

	if systemPrompt != "" {
		contents = append(contents, geminiContent{
			Parts: []geminiPart{{Text: systemPrompt}},
			Role:  "user",
		})
	}

	for _, msg := range messages {
		role := "user"
		if msg.Role == RoleAssistant {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Parts: []geminiPart{{Text: msg.Content}},
			Role:  role,
		})
	}

	reqBody := geminiRequest{
		Contents: contents,
	}
	reqBody.GenerationConfig.MaxOutputTokens = g.config.MaxTokens
	reqBody.GenerationConfig.Temperature = g.config.Temperature

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", g.config.BaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

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

	var geminiResp geminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}

	if len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

func (g *GeminiLLM) Stream(messages []Message, callback func(chunk string)) error {
	contents := make([]geminiContent, len(messages))
	for i, msg := range messages {
		role := "user"
		if msg.Role == RoleAssistant {
			role = "model"
		}
		contents[i] = geminiContent{
			Parts: []geminiPart{{Text: msg.Content}},
			Role:  role,
		}
	}

	streamURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?key=%s", g.config.Model, g.config.APIKey)

	reqBody := geminiRequest{
		Contents: contents,
	}
	reqBody.GenerationConfig.MaxOutputTokens = g.config.MaxTokens
	reqBody.GenerationConfig.Temperature = g.config.Temperature

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", streamURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
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
		var event geminiStreamEvent
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		if len(event.Candidates) > 0 && len(event.Candidates[0].Content.Parts) > 0 {
			callback(event.Candidates[0].Content.Parts[0].Text)
		}
	}

	return nil
}

func (g *GeminiLLM) Close() error {
	return nil
}
