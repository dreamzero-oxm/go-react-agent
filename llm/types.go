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
	APIKey      string  `json:"api_key"`
	BaseURL     string  `json:"base_url"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

type LLM interface {
	Generate(messages []Message) (string, error)
	GenerateWithSystem(systemPrompt string, messages []Message) (string, error)
	Stream(messages []Message, callback func(chunk string)) error
	Close() error
}

type Provider string

const (
	ProviderOpenAI Provider = "openai"
	ProviderCustom Provider = "custom"
)
