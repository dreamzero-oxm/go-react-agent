package llm

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type BedrockLLM struct {
	client *http.Client
	config *LLMConfig
	region string
}

type bedrockMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type bedrockRequest struct {
	MaxTokens   int              `json:"max_gen_len"`
	AnthropicVersion string       `json:"anthropic_version"`
	Messages    []bedrockMessage `json:"messages"`
	Stream      bool             `json:"stream,omitempty"`
}

type bedrockResponse struct {
	Completion string `json:"completion"`
}

type bedrockStreamEvent struct {
	Type  string `json:"type"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
}

func NewBedrockLLM(config *LLMConfig) (*BedrockLLM, error) {
	if config.APIKey == "" || config.SecretAccessKey == "" {
		return nil, fmt.Errorf("AWS Access Key ID and Secret Access Key are required for Bedrock")
	}

	region := config.Region
	if region == "" {
		region = "us-east-1"
	}

	if config.Model == "" {
		config.Model = "anthropic.claude-3-sonnet-20240229-v1:0"
	}

	if config.BaseURL == "" {
		config.BaseURL = fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/model/%s/converse", region, config.Model)
	}

	return &BedrockLLM{
		client: &http.Client{},
		config: config,
		region: region,
	}, nil
}

func (b *BedrockLLM) Generate(messages []Message) (string, error) {
	return b.GenerateWithSystem("", messages)
}

func (b *BedrockLLM) GenerateWithSystem(systemPrompt string, messages []Message) (string, error) {
	bedrockMessages := make([]bedrockMessage, 0, len(messages)+1)

	if systemPrompt != "" {
		bedrockMessages = append(bedrockMessages, bedrockMessage{
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
		bedrockMessages = append(bedrockMessages, bedrockMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	reqBody := bedrockRequest{
		MaxTokens:   b.config.MaxTokens,
		AnthropicVersion: "bedrock-2023-05-31",
		Messages:    bedrockMessages,
		Stream:      false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", b.config.BaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	bedrockReq := b.signAWSRequest(req, jsonBody)

	resp, err := b.client.Do(bedrockReq)
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

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if output, ok := result["output"].(map[string]interface{}); ok {
		if message, ok := output["message"].(map[string]interface{}); ok {
			if content, ok := message["content"].([]interface{}); ok && len(content) > 0 {
				if text, ok := content[0].(map[string]interface{})["text"].(string); ok {
					return text, nil
				}
			}
		}
	}

	return "", fmt.Errorf("unexpected response format")
}

func (b *BedrockLLM) Stream(messages []Message, callback func(chunk string)) error {
	bedrockMessages := make([]bedrockMessage, len(messages))
	for i, msg := range messages {
		role := string(msg.Role)
		if role == "assistant" {
			role = "assistant"
		} else {
			role = "user"
		}
		bedrockMessages[i] = bedrockMessage{
			Role:    role,
			Content: msg.Content,
		}
	}

	reqBody := bedrockRequest{
		MaxTokens:   b.config.MaxTokens,
		AnthropicVersion: "bedrock-2023-05-31",
		Messages:    bedrockMessages,
		Stream:      true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", b.config.BaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	bedrockReq := b.signAWSRequest(req, jsonBody)

	resp, err := b.client.Do(bedrockReq)
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
		var event map[string]interface{}
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		if eventType, ok := event["type"].(string); ok {
			if eventType == "content_block_delta" {
				if delta, ok := event["delta"].(map[string]interface{}); ok {
					if text, ok := delta["text"].(string); ok {
						callback(text)
					}
				}
			}
		}
	}

	return nil
}

func (b *BedrockLLM) signAWSRequest(req *http.Request, body []byte) *http.Request {
	timestamp := time.Now().UTC().Format("20060102T150405Z")
	date := time.Now().UTC().Format("20060102")

	credentialScope := fmt.Sprintf("%s/%s/bedrock/aws4_request", date, b.region)
	algorithm := "AWS4-HMAC-SHA256"

	req.Header.Set("X-Amz-Date", timestamp)
	req.Header.Set("X-Amz-Content-Sha256", hex.EncodeToString(hashSHA256(body)))

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		req.Method,
		req.URL.Path,
		req.URL.Query().Encode(),
		getCanonicalHeaders(req),
		getSignedHeaders(req),
		hex.EncodeToString(hashSHA256(body)),
	)

	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s",
		algorithm,
		timestamp,
		credentialScope,
		hex.EncodeToString(hashSHA256([]byte(canonicalRequest))),
	)

	signingKey := b.getSigningKey(date)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm,
		b.config.AccessKeyID,
		credentialScope,
		getSignedHeaders(req),
		signature,
	)

	req.Header.Set("Authorization", authHeader)
	return req
}

func (b *BedrockLLM) getSigningKey(date string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+b.config.SecretAccessKey), []byte(date))
	kRegion := hmacSHA256(kDate, []byte(b.region))
	kService := hmacSHA256(kRegion, []byte("bedrock"))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}

func hashSHA256(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}

func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

func getCanonicalHeaders(req *http.Request) string {
	headers := ""
	for name, values := range req.Header {
		lowerName := http.CanonicalHeaderKey(name)
		for _, value := range values {
			headers += lowerName + ":" + value + "\n"
		}
	}
	return headers
}

func getSignedHeaders(req *http.Request) string {
	signedHeaders := ""
	for name := range req.Header {
		if signedHeaders != "" {
			signedHeaders += ";"
		}
		signedHeaders += http.CanonicalHeaderKey(name)
	}
	return signedHeaders
}

func (b *BedrockLLM) Close() error {
	return nil
}
