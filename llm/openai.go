package llm

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

type OpenAILLM struct {
	client *openai.Client
	config *LLMConfig
}

func NewOpenAILLM(config *LLMConfig) (*OpenAILLM, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	clientConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}

	return &OpenAILLM{
		client: openai.NewClientWithConfig(clientConfig),
		config: config,
	}, nil
}

func (o *OpenAILLM) Generate(messages []Message) (string, error) {
	return o.GenerateWithSystem("", messages)
}

func (o *OpenAILLM) GenerateWithSystem(systemPrompt string, messages []Message) (string, error) {
	ctx := context.Background()

	openAIMessages := make([]openai.ChatCompletionMessage, 0, len(messages)+1)

	if systemPrompt != "" {
		openAIMessages = append(openAIMessages, openai.ChatCompletionMessage{
			Role:    string(RoleSystem),
			Content: systemPrompt,
		})
	}

	for _, msg := range messages {
		openAIMessages = append(openAIMessages, openai.ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	req := openai.ChatCompletionRequest{
		Model:       o.config.Model,
		Messages:    openAIMessages,
		Temperature: float32(o.config.Temperature),
		MaxTokens:   o.config.MaxTokens,
	}

	resp, err := o.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return resp.Choices[0].Message.Content, nil
}

func (o *OpenAILLM) Stream(messages []Message, callback func(chunk string)) error {
	ctx := context.Background()

	openAIMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		openAIMessages[i] = openai.ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	req := openai.ChatCompletionRequest{
		Model:       o.config.Model,
		Messages:    openAIMessages,
		Temperature: float32(o.config.Temperature),
		MaxTokens:   o.config.MaxTokens,
		Stream:      true,
	}

	stream, err := o.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create chat completion stream: %w", err)
	}
	defer stream.Close()

	for {
		response, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("error receiving stream: %w", err)
		}

		if len(response.Choices) > 0 {
			callback(response.Choices[0].Delta.Content)
		}
	}

	return nil
}

func (o *OpenAILLM) Close() error {
	return nil
}
