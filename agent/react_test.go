package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-react-agent/llm"
	"github.com/go-react-agent/logger"
)

type MockLLMForTest struct {
	responses []string
	index     int
}

func (m *MockLLMForTest) Generate(messages []llm.Message) (string, error) {
	if m.index >= len(m.responses) {
		return "", errors.New("no more responses")
	}
	resp := m.responses[m.index]
	m.index++
	return resp, nil
}

func (m *MockLLMForTest) GenerateWithSystem(systemPrompt string, messages []llm.Message) (string, error) {
	return m.Generate(messages)
}

func (m *MockLLMForTest) Stream(messages []llm.Message, callback func(chunk string)) error {
	return errors.New("not implemented")
}

func (m *MockLLMForTest) Close() error {
	return nil
}

func TestNewReActAgent(t *testing.T) {
	log := logger.NewMultiLogger()
	mockLLM := &MockLLMForTest{}

	agent := NewReActAgent(mockLLM, DefaultConfig(), log)
	if agent == nil {
		t.Fatal("Failed to create agent")
	}
}

func TestReActAgentRegisterTool(t *testing.T) {
	log := logger.NewMultiLogger()
	mockLLM := &MockLLMForTest{}
	agent := NewReActAgent(mockLLM, DefaultConfig(), log)

	testTool := &Tool{
		Name:        "test_tool",
		Description: "Test tool",
		Parameters: map[string]Parameter{
			"input": {Type: "string", Description: "Test input", Required: true},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			return "result", nil
		},
	}

	err := agent.RegisterTool(testTool)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	err = agent.UnregisterTool("test_tool")
	if err != nil {
		t.Fatalf("Failed to unregister tool: %v", err)
	}
}

func TestReActAgentParseResponse(t *testing.T) {
	log := logger.NewMultiLogger()
	log.Disable()
	mockLLM := &MockLLMForTest{}
	agent := NewReActAgent(mockLLM, DefaultConfig(), log)

	testCases := []struct {
		name     string
		response string
		wantDone bool
		wantErr  bool
	}{
		{
			name:     "Answer only",
			response: `{"thoughts":[{"content":"I have the answer"}],"action":null,"answer":"This is the answer","done":true}`,
			wantDone: true,
			wantErr:  false,
		},
		{
			name:     "Thought and Answer",
			response: `{"thoughts":[{"content":"I need to think"}],"action":null,"answer":"Final answer","done":true}`,
			wantDone: true,
			wantErr:  false,
		},
		{
			name:     "Thought and Action",
			response: `{"thoughts":[{"content":"I should use a tool"}],"action":{"name":"test","input":{}},"answer":"","done":false}`,
			wantDone: false,
			wantErr:  false,
		},
		{
			name:     "Invalid JSON",
			response: `Invalid format`,
			wantDone: false,
			wantErr:  true,
		},
		{
			name:     "No thoughts",
			response: `{"thoughts":[],"action":null,"answer":"answer","done":true}`,
			wantDone: false,
			wantErr:  true,
		},
		{
			name:     "Both action and answer",
			response: `{"thoughts":[{"content":"test"}],"action":{"name":"test","input":{}},"answer":"answer","done":false}`,
			wantDone: false,
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := agent.parseResponse(tc.response)
			if tc.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tc.wantErr && parsed.Done != tc.wantDone {
				t.Errorf("Expected done=%v, got %v", tc.wantDone, parsed.Done)
			}
		})
	}
}

func TestReActAgentRun(t *testing.T) {
	log := logger.NewMultiLogger()
	log.Disable()

	mockLLM := &MockLLMForTest{
		responses: []string{
			`{"thoughts":[{"content":"I have the answer"}],"action":null,"answer":"The answer is 42","done":true}`,
		},
	}

	agent := NewReActAgent(mockLLM, DefaultConfig(), log)

	response, err := agent.Run(context.Background(), "What is the answer?")
	if err != nil {
		t.Fatalf("Failed to run agent: %v", err)
	}

	if !response.Done {
		t.Error("Expected response to be done")
	}

	if response.Answer != "The answer is 42" {
		t.Errorf("Expected answer 'The answer is 42', got '%s'", response.Answer)
	}
}

func TestReActAgentRunWithTools(t *testing.T) {
	log := logger.NewMultiLogger()
	log.Disable()

	mockLLM := &MockLLMForTest{
		responses: []string{
			`{"thoughts":[{"content":"I need to use a tool"}],"action":{"name":"echo","input":{"text":"hello"}},"answer":"","done":false}`,
			`{"thoughts":[{"content":"Based on the tool result, I can now answer"}],"action":null,"answer":"The tool returned hello","done":true}`,
		},
	}

	agent := NewReActAgent(mockLLM, DefaultConfig(), log)

	echoTool := &Tool{
		Name:        "echo",
		Description: "Echo text",
		Execute: func(input map[string]interface{}) (string, error) {
			text, ok := input["text"].(string)
			if !ok {
				return "", errors.New("text must be string")
			}
			return text, nil
		},
	}

	err := agent.RegisterTool(echoTool)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	response, err := agent.Run(context.Background(), "Say hello")
	if err != nil {
		t.Fatalf("Failed to run agent: %v", err)
	}

	if !response.Done {
		t.Error("Expected response to be done")
	}

	if len(response.Thoughts) == 0 {
		t.Error("Expected at least one thought")
	}
}

func TestReActAgentRunWithCallback(t *testing.T) {
	log := logger.NewMultiLogger()
	log.Disable()

	mockLLM := &MockLLMForTest{
		responses: []string{
			`{"thoughts":[{"content":"Using echo tool"}],"action":{"name":"echo","input":{"text":"test"}},"answer":"","done":false}`,
			`{"thoughts":[{"content":"Done"}],"action":null,"answer":"Complete","done":true}`,
		},
	}

	agent := NewReActAgent(mockLLM, DefaultConfig(), log)

	echoTool := &Tool{
		Name:        "echo",
		Description: "Echo text",
		Execute: func(input map[string]interface{}) (string, error) {
			text, _ := input["text"].(string)
			return text, nil
		},
	}

	agent.RegisterTool(echoTool)

	callbackCalled := false
	var callbackStep *Step

	response, err := agent.RunWithCallback(context.Background(), "Test", func(step *Step) {
		callbackCalled = true
		callbackStep = step
	})

	if err != nil {
		t.Fatalf("Failed to run agent with callback: %v", err)
	}

	if !callbackCalled {
		t.Error("Callback was not called")
	}

	if callbackStep == nil {
		t.Error("Callback step is nil")
	}

	if !response.Done {
		t.Error("Expected response to be done")
	}
}

func TestReActAgentMaxIterations(t *testing.T) {
	log := logger.NewMultiLogger()
	log.Disable()

	responses := make([]string, 20)
	for i := 0; i < 20; i++ {
		responses[i] = `{"thoughts":[{"content":"Thinking..."}],"action":{"name":"echo","input":{"text":"test"}},"answer":"","done":false}`
	}

	mockLLM := &MockLLMForTest{responses: responses}

	config := DefaultConfig()
	config.MaxIterations = 5

	agent := NewReActAgent(mockLLM, config, log)

	echoTool := &Tool{
		Name:        "echo",
		Description: "Echo text",
		Execute: func(input map[string]interface{}) (string, error) {
			return "test", nil
		},
	}

	agent.RegisterTool(echoTool)

	_, err := agent.Run(context.Background(), "Test")
	if err == nil {
		t.Error("Expected error for max iterations")
	}

	if !contains(err.Error(), "max iterations") {
		t.Errorf("Expected max iterations error, got: %v", err)
	}
}

func TestReActAgentTimeout(t *testing.T) {
	log := logger.NewMultiLogger()
	log.Disable()

	mockLLM := &MockLLMForTest{
		responses: []string{
			`{"thoughts":[{"content":"Thinking..."}],"action":null,"answer":"","done":false}`,
		},
	}

	config := DefaultConfig()
	config.Timeout = 100 * time.Millisecond

	agent := NewReActAgent(mockLLM, config, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	_, err := agent.Run(ctx, "Test")
	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestReActAgentSystemPrompt(t *testing.T) {
	log := logger.NewMultiLogger()
	log.Disable()

	mockLLM := &MockLLMForTest{}

	agent := NewReActAgent(mockLLM, DefaultConfig(), log)

	customPrompt := "You are a custom agent"
	agent.SetSystemPrompt(customPrompt)

	gotPrompt := agent.GetSystemPrompt()
	targetPrompt := customPrompt + "\n\n## Available Tools\n\n"
	if gotPrompt != targetPrompt {
		t.Errorf("Expected custom prompt to end with proper format, got:\n%s\n\nexpected to end with:\n%s", gotPrompt, targetPrompt)
	}
}

func TestReActAgentSystemPromptWithToolPlaceholder(t *testing.T) {
	log := logger.NewMultiLogger()
	log.Disable()

	mockLLM := &MockLLMForTest{}

	agent := NewReActAgent(mockLLM, DefaultConfig(), log)

	customPrompt := "You are a helpful assistant.\n\nAvailable tools:\n{{tools}}"
	agent.SetSystemPrompt(customPrompt)
	
	echoTool := &Tool{
		Name:        "echo",
		Description: "Echo text",
		Execute: func(input map[string]interface{}) (string, error) {
			text, _ := input["text"].(string)
			return text, nil
		},
	}
	agent.RegisterTool(echoTool)

	gotPrompt := agent.GetSystemPrompt()
	if !strings.Contains(gotPrompt, "Available tools:") {
		t.Errorf("Expected prompt to contain 'Available tools:', got '%s'", gotPrompt)
	}
	if !strings.Contains(gotPrompt, "echo") {
		t.Errorf("Expected prompt to contain tool 'echo', got '%s'", gotPrompt)
	}
	if strings.Contains(gotPrompt, "{{tools}}") {
		t.Errorf("Expected prompt to not contain placeholder '{{tools}}', got '%s'", gotPrompt)
	}
}

func TestReActAgentSystemPromptWithCustomPlaceholder(t *testing.T) {
	log := logger.NewMultiLogger()
	log.Disable()

	mockLLM := &MockLLMForTest{}

	agent := NewReActAgent(mockLLM, DefaultConfig(), log)

	echoTool := &Tool{
		Name:        "echo",
		Description: "Echo text",
		Execute: func(input map[string]interface{}) (string, error) {
			text, _ := input["text"].(string)
			return text, nil
		},
	}
	agent.RegisterTool(echoTool)

	customPrompt := "You are a helpful assistant.\n\n{{TOOLS}}"
	agent.SetSystemPrompt(customPrompt)

	gotPrompt := agent.GetSystemPrompt()
	if !strings.Contains(gotPrompt, "echo") {
		t.Errorf("Expected prompt to contain tool 'echo', got '%s'", gotPrompt)
	}
	if strings.Contains(gotPrompt, "{{TOOLS}}") {
		t.Errorf("Expected prompt to not contain placeholder '{{TOOLS}}', got '%s'", gotPrompt)
	}
}

func TestReActAgentDefaultPromptHasTools(t *testing.T) {
	log := logger.NewMultiLogger()
	log.Disable()

	mockLLM := &MockLLMForTest{}

	agent := NewReActAgent(mockLLM, DefaultConfig(), log)

	echoTool := &Tool{
		Name:        "echo",
		Description: "Echo text",
		Execute: func(input map[string]interface{}) (string, error) {
			text, _ := input["text"].(string)
			return text, nil
		},
	}
	agent.RegisterTool(echoTool)

	gotPrompt := agent.GetSystemPrompt()
	if !strings.Contains(gotPrompt, "Available tools:") {
		t.Errorf("Expected default prompt to contain 'Available tools:', got '%s'", gotPrompt)
	}
	if !strings.Contains(gotPrompt, "echo") {
		t.Errorf("Expected default prompt to contain tool 'echo', got '%s'", gotPrompt)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MaxIterations != 10 {
		t.Errorf("Expected MaxIterations 10, got %d", config.MaxIterations)
	}

	if config.Temperature != 0.7 {
		t.Errorf("Expected Temperature 0.7, got %f", config.Temperature)
	}

	if config.MaxTokens != 2000 {
		t.Errorf("Expected MaxTokens 2000, got %d", config.MaxTokens)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
