package llm

import (
	"fmt"
)

type LLMFactory interface {
	CreateLLM(config *LLMConfig) (LLM, error)
}

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

func NewLLMWithProvider(provider Provider, apiKey string, model string) (LLM, error) {
	config := &LLMConfig{
		APIKey:   apiKey,
		Model:    model,
		Provider: provider,
	}
	return NewLLM(config)
}
