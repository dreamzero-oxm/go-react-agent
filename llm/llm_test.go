package llm

import (
	"testing"
)

type MockLLMForTest struct {
	responses []string
	index     int
}

func (m *MockLLMForTest) Generate(messages []Message) (string, error) {
	if m.index >= len(m.responses) {
		return "", nil
	}
	resp := m.responses[m.index]
	m.index++
	return resp, nil
}

func (m *MockLLMForTest) GenerateWithSystem(systemPrompt string, messages []Message) (string, error) {
	return m.Generate(messages)
}

func (m *MockLLMForTest) Stream(messages []Message, callback func(chunk string)) error {
	return nil
}

func (m *MockLLMForTest) Close() error {
	return nil
}

func TestLLMConfig(t *testing.T) {
	config := &LLMConfig{
		APIKey:      "test-key",
		BaseURL:     "https://api.example.com",
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	if config.APIKey != "test-key" {
		t.Errorf("Expected APIKey 'test-key', got '%s'", config.APIKey)
	}

	if config.Model != "gpt-4" {
		t.Errorf("Expected Model 'gpt-4', got '%s'", config.Model)
	}
}

func TestMockLLM(t *testing.T) {
	mock := &MockLLMForTest{
		responses: []string{"response1", "response2"},
	}

	messages := []Message{
		{Role: RoleUser, Content: "test"},
	}

	resp1, err := mock.Generate(messages)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if resp1 != "response1" {
		t.Errorf("Expected 'response1', got '%s'", resp1)
	}

	resp2, err := mock.Generate(messages)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if resp2 != "response2" {
		t.Errorf("Expected 'response2', got '%s'", resp2)
	}
}

func TestProvider(t *testing.T) {
	if ProviderOpenAI != "openai" {
		t.Errorf("Expected ProviderOpenAI 'openai', got '%s'", ProviderOpenAI)
	}

	if ProviderCustom != "custom" {
		t.Errorf("Expected ProviderCustom 'custom', got '%s'", ProviderCustom)
	}
}
