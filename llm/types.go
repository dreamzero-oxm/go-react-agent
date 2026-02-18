package llm

type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
	RoleTool      MessageRole = "tool"
)

type Message struct {
	Role    MessageRole `json:"role"`
	Content string      `json:"content"`
}

type LLMConfig struct {
	APIKey          string   `json:"api_key"`
	BaseURL         string   `json:"base_url"`
	Model           string   `json:"model"`
	Temperature     float64  `json:"temperature"`
	MaxTokens       int      `json:"max_tokens"`
	Provider        Provider `json:"provider"`
	Stream          bool     `json:"stream"`
	EnableStreaming bool     `json:"enable_streaming"`
	Region          string   `json:"region,omitempty"`
	ProjectID       string   `json:"project_id,omitempty"`
	AccessKeyID     string   `json:"access_key_id,omitempty"`
	SecretAccessKey string   `json:"secret_access_key,omitempty"`
	SessionToken    string   `json:"session_token,omitempty"`
}

type LLM interface {
	Generate(messages []Message) (string, error)
	GenerateWithSystem(systemPrompt string, messages []Message) (string, error)
	Stream(messages []Message, callback func(chunk string)) error
	Close() error
}

type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderCustom    Provider = "custom"
	ProviderAnthropic Provider = "anthropic"
	ProviderGemini    Provider = "gemini"
	ProviderCohere    Provider = "cohere"
	ProviderMistral   Provider = "mistral"
	ProviderBedrock   Provider = "bedrock"
	ProviderDashScope Provider = "dashscope"
	ProviderWenxin    Provider = "wenxin"
	ProviderOllama    Provider = "ollama"
	ProviderGeneric   Provider = "generic"
)
