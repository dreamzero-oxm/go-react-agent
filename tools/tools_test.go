package tools

import (
	"fmt"
	"testing"
)

func TestToolRegistry(t *testing.T) {
	registry := NewToolRegistry()

	testTool := &Tool{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters: map[string]Parameter{
			"input": {
				Type:        "string",
				Description: "Test input",
				Required:    true,
			},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			return "executed", nil
		},
	}

	err := registry.RegisterTool(testTool)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	tool, err := registry.Get("test_tool")
	if err != nil {
		t.Fatalf("Failed to get tool: %v", err)
	}

	if tool.Name != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", tool.Name)
	}

	err = registry.UnregisterTool("test_tool")
	if err != nil {
		t.Fatalf("Failed to unregister tool: %v", err)
	}

	_, err = registry.Get("test_tool")
	if err == nil {
		t.Error("Expected error when getting unregistered tool")
	}
}

func TestToolRegistryDuplicate(t *testing.T) {
	registry := NewToolRegistry()

	testTool := &Tool{
		Name:        "duplicate_tool",
		Description: "A test tool",
		Execute:     func(input map[string]interface{}) (string, error) { return "", nil },
	}

	err := registry.RegisterTool(testTool)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	err = registry.RegisterTool(testTool)
	if err == nil {
		t.Error("Expected error when registering duplicate tool")
	}
}

func TestToolRegistryExecute(t *testing.T) {
	registry := NewToolRegistry()

	testTool := &Tool{
		Name:        "echo_tool",
		Description: "Echo tool",
		Execute: func(input map[string]interface{}) (string, error) {
			text, ok := input["text"].(string)
			if !ok {
				return "", NewParameterError("text must be a string")
			}
			return text, nil
		},
	}

	registry.RegisterTool(testTool)

	result, err := registry.Execute("echo_tool", map[string]interface{}{"text": "hello"})
	if err != nil {
		t.Fatalf("Failed to execute tool: %v", err)
	}

	if result != "hello" {
		t.Errorf("Expected result 'hello', got '%s'", result)
	}

	_, err = registry.Execute("echo_tool", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for missing required parameter")
	}
}

func TestToolRegistryList(t *testing.T) {
	registry := NewToolRegistry()

	registry.RegisterTool(&Tool{Name: "tool1", Execute: func(input map[string]interface{}) (string, error) { return "", nil }})
	registry.RegisterTool(&Tool{Name: "tool2", Execute: func(input map[string]interface{}) (string, error) { return "", nil }})
	registry.RegisterTool(&Tool{Name: "tool3", Execute: func(input map[string]interface{}) (string, error) { return "", nil }})

	tools := registry.List()
	if len(tools) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(tools))
	}

	for _, toolName := range []string{"tool1", "tool2", "tool3"} {
		found := false
		for _, name := range tools {
			if name == toolName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Tool '%s' not found in list", toolName)
		}
	}
}

func TestToolRegistrySchema(t *testing.T) {
	registry := NewToolRegistry()

	registry.RegisterTool(&Tool{
		Name:        "test_tool",
		Description: "Test description",
		Parameters: map[string]Parameter{
			"param1": {Type: "string", Description: "Param 1", Required: true},
		},
		Execute: func(input map[string]interface{}) (string, error) { return "", nil },
	})

	schema := registry.GetToolsSchema()
	if len(schema) == 0 {
		t.Error("Schema is empty")
	}

	expectedSubstrings := []string{"test_tool", "Test description", "param1", "string", "required"}
	for _, substr := range expectedSubstrings {
		if !contains(schema, substr) {
			t.Errorf("Schema missing expected substring '%s'", substr)
		}
	}
}

func TestBuiltinTools(t *testing.T) {
	registry := NewToolRegistry()
	RegisterBuiltinTools(registry)

	tools := registry.List()
	if len(tools) == 0 {
		t.Error("No builtin tools registered")
	}

	for _, toolName := range []string{"calculate", "echo"} {
		_, err := registry.Get(toolName)
		if err != nil {
			t.Errorf("Builtin tool '%s' not found", toolName)
		}
	}
}

func TestEchoTool(t *testing.T) {
	registry := NewToolRegistry()
	RegisterBuiltinTools(registry)

	result, err := registry.Execute("echo", map[string]interface{}{"text": "test message"})
	if err != nil {
		t.Fatalf("Failed to execute echo tool: %v", err)
	}

	if result != "test message" {
		t.Errorf("Expected 'test message', got '%s'", result)
	}
}

func TestCalculateTool(t *testing.T) {
	registry := NewToolRegistry()
	RegisterBuiltinTools(registry)

	result, err := registry.Execute("calculate", map[string]interface{}{"expression": "2 + 3"})
	if err != nil {
		t.Fatalf("Failed to execute calculate tool: %v", err)
	}

	expected := "Result: 2 + 3 = 5"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestCalculateToolSimple(t *testing.T) {
	registry := NewToolRegistry()
	RegisterBuiltinTools(registry)

	testCases := []struct {
		expr     string
		expected float64
	}{
		{"10 - 3", 7},
		{"4 * 5", 20},
		{"20 / 4", 5},
	}

	for _, tc := range testCases {
		t.Run(tc.expr, func(t *testing.T) {
			result, err := registry.Execute("calculate", map[string]interface{}{"expression": tc.expr})
			if err != nil {
				t.Fatalf("Failed to execute calculate tool: %v", err)
			}

			expectedStr := "Result: " + tc.expr + " = " + formatFloat(tc.expected)
			if result != expectedStr {
				t.Errorf("Expected '%s', got '%s'", expectedStr, result)
			}
		})
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

type ParameterError struct {
	message string
}

func NewParameterError(msg string) *ParameterError {
	return &ParameterError{message: msg}
}

func (e *ParameterError) Error() string {
	return e.message
}

func formatFloat(f float64) string {
	if f == float64(int(f)) {
		result := int(f)
		return fmt.Sprintf("%d", result)
	}
	return fmt.Sprintf("%.0f", f)
}
