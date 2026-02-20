package llm

import (
	"fmt"
)

// LLMFactory defines the interface for creating LLM instances.
type LLMFactory interface {
	// CreateLLM creates a new LLM instance from the given configuration.
	CreateLLM(config *LLMConfig) (LLM, error)
}

// NewLLM creates a new LLM instance based on the provider in the configuration.
//
// It supports the following providers: openai, anthropic, gemini, cohere, mistral,
// bedrock, dashscope, wenxin, ollama, generic, and custom.
func NewLLM(config *LLMConfig) (LLM, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	switch config.Provider {
	case ProviderOpenAI:
		return NewOpenAILLM(config)
	case ProviderAnthropic:
		return NewAnthropicLLM(config)
	case ProviderGemini:
		return NewGeminiLLM(config)
	case ProviderCohere:
		return NewCohereLLM(config)
	case ProviderMistral:
		return NewMistralLLM(config)
	case ProviderBedrock:
		return NewBedrockLLM(config)
	case ProviderDashScope:
		return NewDashScopeLLM(config)
	case ProviderWenxin:
		return NewWenxinLLM(config)
	case ProviderOllama:
		return NewOllamaLLM(config)
	case ProviderGeneric:
		return NewGenericLLM(config)
	case ProviderCustom:
		return NewCustomLLM(config)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}
}

// NewLLMWithProvider creates a new LLM instance with the specified provider, API key, and model.
//
// This is a convenience function for quickly creating LLM instances without
// constructing a full LLMConfig.
func NewLLMWithProvider(provider Provider, apiKey string, model string) (LLM, error) {
	config := &LLMConfig{
		APIKey:   apiKey,
		Model:    model,
		Provider: provider,
	}
	return NewLLM(config)
}
