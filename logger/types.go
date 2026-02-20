// Package logger provides a flexible multi-target logging system with support
// for console output, file logging, and external logger integration.
package logger

// LogLevel represents the severity level of a log message.
type LogLevel int

const (
	// LevelDebug is used for debug-level messages.
	LevelDebug LogLevel = iota
	// LevelInfo is used for informational messages.
	LevelInfo
	// LevelWarn is used for warning messages.
	LevelWarn
	// LevelError is used for error messages.
	LevelError
	// LevelFatal is used for fatal messages (followed by exit).
	LevelFatal
)

// LogEntry represents a single log entry.
type LogEntry struct {
	// Level is the severity level of the log entry
	Level LogLevel
	// Message is the log message text
	Message string
	// Fields contains additional structured data
	Fields map[string]interface{}
	// Time is the timestamp of the log entry
	Time string
}

// Logger defines the interface for logging operations.
type Logger interface {
	// Debug logs a debug-level message.
	Debug(msg string, fields map[string]interface{})
	// Info logs an informational message.
	Info(msg string, fields map[string]interface{})
	// Warn logs a warning message.
	Warn(msg string, fields map[string]interface{})
	// Error logs an error message.
	Error(msg string, fields map[string]interface{})
	// Fatal logs a fatal message and exits the program.
	Fatal(msg string, fields map[string]interface{})
	// SetLevel sets the minimum log level.
	SetLevel(level LogLevel)
	// Close closes the logger and releases resources.
	Close() error
}
