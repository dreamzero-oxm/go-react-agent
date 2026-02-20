// Package llm provides a unified interface for interacting with various
// Large Language Model providers including OpenAI, Anthropic, Google Gemini,
// and others.
package llm

// MessageRole represents the role of a message sender in a conversation.
type MessageRole string

const (
	// RoleUser represents messages from the user.
	RoleUser MessageRole = "user"
	// RoleAssistant represents messages from the AI assistant.
	RoleAssistant MessageRole = "assistant"
	// RoleSystem represents system messages that set behavior.
	RoleSystem MessageRole = "system"
	// RoleTool represents messages from tool execution results.
	RoleTool MessageRole = "tool"
)

// Message represents a single message in a conversation.
type Message struct {
	// Role is the sender of the message (user, assistant, system, tool)
	Role MessageRole `json:"role"`
	// Content is the text content of the message
	Content string `json:"content"`
}

// LLMConfig holds configuration for LLM providers.
type LLMConfig struct {
	// APIKey is the authentication key for the LLM provider
	APIKey string `json:"api_key"`
	// BaseURL is the base URL for the LLM API (for custom endpoints)
	BaseURL string `json:"base_url"`
	// Model is the name of the model to use
	Model string `json:"model"`
	// Temperature controls response randomness (0.0 to 1.0)
	Temperature float64 `json:"temperature"`
	// MaxTokens is the maximum number of tokens in the response
	MaxTokens int `json:"max_tokens"`
	// Provider is the LLM provider to use
	Provider Provider `json:"provider"`
	// Stream enables streaming responses
	Stream bool `json:"stream"`
	// EnableStreaming is an alias for Stream
	EnableStreaming bool `json:"enable_streaming"`
	// Region is the AWS region for Bedrock
	Region string `json:"region,omitempty"`
	// ProjectID is the Google Cloud project ID for Gemini
	ProjectID string `json:"project_id,omitempty"`
	// AccessKeyID is the AWS access key ID for Bedrock
	AccessKeyID string `json:"access_key_id,omitempty"`
	// SecretAccessKey is the AWS secret access key for Bedrock
	SecretAccessKey string `json:"secret_access_key,omitempty"`
	// SessionToken is the AWS session token for Bedrock
	SessionToken string `json:"session_token,omitempty"`
}

// LLM defines the interface for LLM providers.
type LLM interface {
	// Generate generates a response from the given messages.
	Generate(messages []Message) (string, error)
	// GenerateWithSystem generates a response with a system prompt.
	GenerateWithSystem(systemPrompt string, messages []Message) (string, error)
	// Stream generates a streaming response with a callback for each chunk.
	Stream(messages []Message, callback func(chunk string)) error
	// Close closes the LLM connection and releases resources.
	Close() error
}

// Provider represents an LLM provider.
type Provider string

const (
	// ProviderOpenAI is the OpenAI provider.
	ProviderOpenAI Provider = "openai"
	// ProviderCustom is a custom provider.
	ProviderCustom Provider = "custom"
	// ProviderAnthropic is the Anthropic provider.
	ProviderAnthropic Provider = "anthropic"
	// ProviderGemini is the Google Gemini provider.
	ProviderGemini Provider = "gemini"
	// ProviderCohere is the Cohere provider.
	ProviderCohere Provider = "cohere"
	// ProviderMistral is the Mistral AI provider.
	ProviderMistral Provider = "mistral"
	// ProviderBedrock is the AWS Bedrock provider.
	ProviderBedrock Provider = "bedrock"
	// ProviderDashScope is the Alibaba Cloud DashScope provider.
	ProviderDashScope Provider = "dashscope"
	// ProviderWenxin is the Baidu Wenxin provider.
	ProviderWenxin Provider = "wenxin"
	// ProviderOllama is the Ollama local provider.
	ProviderOllama Provider = "ollama"
	// ProviderGeneric is a generic REST API provider.
	ProviderGeneric Provider = "generic"
)
