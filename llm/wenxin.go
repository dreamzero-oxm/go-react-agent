package llm

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

type WenxinLLM struct {
	client *http.Client
	config *LLMConfig
	accessToken string
}

type wenxinMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type wenxinRequest struct {
	Messages []wenxinMessage `json:"messages"`
	Stream   bool           `json:"stream,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	TopP       float64       `json:"top_p,omitempty"`
}

type wenxinResponse struct {
	Result string `json:"result"`
}

type wenxinStreamEvent struct {
	Result string `json:"result"`
}

func NewWenxinLLM(config *LLMConfig) (*WenxinLLM, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API Key and Secret Key are required for Wenxin (format: APIKey|SecretKey)")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/chat/completions"
	}

	if config.Model == "" {
		config.Model = "ERNIE-Bot-4"
	}

	parts := strings.Split(config.APIKey, "|")
	if len(parts) != 2 {
		return nil, fmt.Errorf("API key must be in format: APIKey|SecretKey")
	}

	apiKey, secretKey := parts[0], parts[1]

	llm := &WenxinLLM{
		client: &http.Client{},
		config: config,
	}

	token, err := llm.getAccessToken(apiKey, secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	llm.accessToken = token
	return llm, nil
}

func (w *WenxinLLM) getAccessToken(apiKey, secretKey string) (string, error) {
	params := url.Values{}
	params.Set("grant_type", "client_credentials")
	params.Set("client_id", apiKey)
	params.Set("client_secret", secretKey)

	req, err := http.NewRequest("POST", "https://aip.baidubce.com/oauth/2.0/token", strings.NewReader(params.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := w.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if token, ok := result["access_token"].(string); ok {
		return token, nil
	}

	return "", fmt.Errorf("no access token in response")
}

func (w *WenxinLLM) Generate(messages []Message) (string, error) {
	return w.GenerateWithSystem("", messages)
}

func (w *WenxinLLM) GenerateWithSystem(systemPrompt string, messages []Message) (string, error) {
	wenxinMessages := make([]wenxinMessage, 0, len(messages)+1)

	if systemPrompt != "" {
		wenxinMessages = append(wenxinMessages, wenxinMessage{
			Role:    "user",
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
		wenxinMessages = append(wenxinMessages, wenxinMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	reqBody := wenxinRequest{
		Messages:    wenxinMessages,
		Stream:      false,
		Temperature: w.config.Temperature,
		TopP:        0.8,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	reqURL, _ := url.Parse(w.config.BaseURL)
	query := reqURL.Query()
	query.Set("access_token", w.accessToken)
	reqURL.RawQuery = query.Encode()

	req, err := http.NewRequest("POST", reqURL.String(), bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
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

	var wenxinResp wenxinResponse
	if err := json.Unmarshal(body, &wenxinResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return wenxinResp.Result, nil
}

func (w *WenxinLLM) Stream(messages []Message, callback func(chunk string)) error {
	wenxinMessages := make([]wenxinMessage, len(messages))
	for i, msg := range messages {
		role := string(msg.Role)
		if role == "assistant" {
			role = "assistant"
		} else {
			role = "user"
		}
		wenxinMessages[i] = wenxinMessage{
			Role:    role,
			Content: msg.Content,
		}
	}

	reqBody := wenxinRequest{
		Messages:    wenxinMessages,
		Stream:      true,
		Temperature: w.config.Temperature,
		TopP:        0.8,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	reqURL, _ := url.Parse(w.config.BaseURL)
	query := reqURL.Query()
	query.Set("access_token", w.accessToken)
	reqURL.RawQuery = query.Encode()

	req, err := http.NewRequest("POST", reqURL.String(), bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
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

		var event wenxinStreamEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		if event.Result != "" {
			callback(event.Result)
		}
	}

	return nil
}

func (w *WenxinLLM) Close() error {
	return nil
}

func generateSignature(params map[string]string, secretKey string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for _, k := range keys {
		if builder.Len() > 0 {
			builder.WriteString("&")
		}
		builder.WriteString(k)
		builder.WriteString("=")
		builder.WriteString(params[k])
	}

	signStr := builder.String() + secretKey

	h := hmac.New(md5.New, []byte(secretKey))
	h.Write([]byte(signStr))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
