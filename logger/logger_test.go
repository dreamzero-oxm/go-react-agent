package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMultiLogger(t *testing.T) {
	ml := NewMultiLogger()
	ml.SetLevel(LevelDebug)

	ml.Debug("Test debug message", map[string]interface{}{"key": "value"})
	ml.Info("Test info message", nil)
	ml.Warn("Test warn message", map[string]interface{}{"warning": true})
	ml.Error("Test error message", map[string]interface{}{"error_code": 500})
}

func TestConsoleLogger(t *testing.T) {
	cl := NewConsoleLogger(false)
	cl.SetLevel(LevelDebug)

	cl.Debug("Debug message", nil)
	cl.Info("Info message", map[string]interface{}{"count": 42})
	cl.Warn("Warn message", nil)
	cl.Error("Error message", map[string]interface{}{"err": "test error"})

	if err := cl.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestConsoleLoggerColor(t *testing.T) {
	cl := NewConsoleLogger(true)
	cl.SetLevel(LevelInfo)

	cl.Info("Colored info message", nil)
	cl.Warn("Colored warn message", nil)
	cl.Error("Colored error message", nil)

	if err := cl.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestFileLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	fl, err := NewFileLogger(logPath)
	if err != nil {
		t.Fatalf("Failed to create file logger: %v", err)
	}

	fl.SetLevel(LevelDebug)
	fl.Debug("Debug message", map[string]interface{}{"debug": true})
	fl.Info("Info message", nil)
	fl.Warn("Warn message", map[string]interface{}{"warn": true})
	fl.Error("Error message", map[string]interface{}{"error": "test"})

	if err := fl.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if len(logContent) == 0 {
		t.Error("Log file is empty")
	}

	expectedStrings := []string{"Debug message", "Info message", "Warn message", "Error message"}
	for _, expected := range expectedStrings {
		if !contains(logContent, expected) {
			t.Errorf("Expected log to contain '%s'", expected)
		}
	}
}

func TestMultiLoggerWithExternal(t *testing.T) {
	ml := NewMultiLogger()
	ml.SetLevel(LevelInfo)

	mockExternal := &mockLogger{}
	ml.SetExternalLogger(mockExternal)

	ml.Info("Test message", map[string]interface{}{"test": true})

	if !mockExternal.lastCalled {
		t.Error("External logger was not called")
	}
}

func TestMultiLoggerDisable(t *testing.T) {
	ml := NewMultiLogger()
	ml.AddConsoleLogger(false)

	ml.Disable()
	ml.Info("This should not be logged", nil)

	ml.Enable()
	ml.Info("This should be logged", nil)

	if err := ml.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestMultiLoggerMultipleOutputs(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "multi.log")

	ml := NewMultiLogger()
	ml.SetLevel(LevelDebug)
	ml.AddConsoleLogger(false)
	_, err := ml.AddFileLogger(logPath)
	if err != nil {
		t.Fatalf("Failed to add file logger: %v", err)
	}

	ml.Info("Multi-output test", map[string]interface{}{"output": "test"})

	if err := ml.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Log file is empty")
	}
}

func TestLogLevelFiltering(t *testing.T) {
	cl := NewConsoleLogger(false)
	cl.SetLevel(LevelWarn)

	cl.Debug("Should not appear", nil)
	cl.Info("Should not appear", nil)
	cl.Warn("Should appear", nil)
	cl.Error("Should appear", nil)

	if err := cl.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

type mockLogger struct {
	lastCalled bool
}

func (m *mockLogger) Debug(msg string, fields map[string]interface{}) {
	m.lastCalled = true
}

func (m *mockLogger) Info(msg string, fields map[string]interface{}) {
	m.lastCalled = true
}

func (m *mockLogger) Warn(msg string, fields map[string]interface{}) {
	m.lastCalled = true
}

func (m *mockLogger) Error(msg string, fields map[string]interface{}) {
	m.lastCalled = true
}

func (m *mockLogger) Fatal(msg string, fields map[string]interface{}) {
	m.lastCalled = true
}

func (m *mockLogger) SetLevel(level LogLevel) {
}

func (m *mockLogger) Close() error {
	return nil
}
