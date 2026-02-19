package logger

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type MultiLogger struct {
	loggers     []Logger
	level       LogLevel
	mu          sync.RWMutex
	disabled    bool
	externalLog Logger
}

func NewMultiLogger() *MultiLogger {
	return &MultiLogger{
		loggers: make([]Logger, 0),
		level:   LevelInfo,
	}
}

func (ml *MultiLogger) AddConsoleLogger(enableColor bool) *ConsoleLogger {
	cl := NewConsoleLogger(enableColor)
	ml.mu.Lock()
	ml.loggers = append(ml.loggers, cl)
	ml.mu.Unlock()
	return cl
}

func (ml *MultiLogger) AddFileLogger(filePath string) (*FileLogger, error) {
	fl, err := NewFileLogger(filePath)
	if err != nil {
		return nil, err
	}
	ml.mu.Lock()
	ml.loggers = append(ml.loggers, fl)
	ml.mu.Unlock()
	return fl, nil
}

func (ml *MultiLogger) SetExternalLogger(logger Logger) {
	ml.mu.Lock()
	ml.externalLog = logger
	ml.mu.Unlock()
}

func (ml *MultiLogger) Disable() {
	ml.mu.Lock()
	ml.disabled = true
	ml.mu.Unlock()
}

func (ml *MultiLogger) Enable() {
	ml.mu.Lock()
	ml.disabled = false
	ml.mu.Unlock()
}

func (ml *MultiLogger) SetLevel(level LogLevel) {
	ml.mu.Lock()
	ml.level = level
	ml.mu.Unlock()
	for _, log := range ml.loggers {
		log.SetLevel(level)
	}
	if ml.externalLog != nil {
		ml.externalLog.SetLevel(level)
	}
}

func (ml *MultiLogger) Debug(msg string, fields map[string]interface{}) {
	ml.log(LevelDebug, msg, fields)
}

func (ml *MultiLogger) Info(msg string, fields map[string]interface{}) {
	ml.log(LevelInfo, msg, fields)
}

func (ml *MultiLogger) Warn(msg string, fields map[string]interface{}) {
	ml.log(LevelWarn, msg, fields)
}

func (ml *MultiLogger) Error(msg string, fields map[string]interface{}) {
	ml.log(LevelError, msg, fields)
}

func (ml *MultiLogger) Fatal(msg string, fields map[string]interface{}) {
	ml.log(LevelFatal, msg, fields)
	os.Exit(1)
}

func (ml *MultiLogger) log(level LogLevel, msg string, fields map[string]interface{}) {
	ml.mu.RLock()
	if ml.disabled || level < ml.level {
		ml.mu.RUnlock()
		return
	}
	ml.mu.RUnlock()

	if ml.externalLog != nil {
		switch level {
		case LevelDebug:
			ml.externalLog.Debug(msg, fields)
		case LevelInfo:
			ml.externalLog.Info(msg, fields)
		case LevelWarn:
			ml.externalLog.Warn(msg, fields)
		case LevelError:
			ml.externalLog.Error(msg, fields)
		case LevelFatal:
			ml.externalLog.Fatal(msg, fields)
		}
		return
	}

	ml.mu.RLock()
	loggers := append([]Logger{}, ml.loggers...)
	ml.mu.RUnlock()

	for _, log := range loggers {
		switch level {
		case LevelDebug:
			log.Debug(msg, fields)
		case LevelInfo:
			log.Info(msg, fields)
		case LevelWarn:
			log.Warn(msg, fields)
		case LevelError:
			log.Error(msg, fields)
		case LevelFatal:
			log.Fatal(msg, fields)
		}
	}
}

func (ml *MultiLogger) Close() error {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	var errs []string
	for _, log := range ml.loggers {
		if err := log.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing loggers: %s", strings.Join(errs, "; "))
	}
	return nil
}

type ConsoleLogger struct {
	level       LogLevel
	enableColor bool
	mu          sync.RWMutex
}

func NewConsoleLogger(enableColor bool) *ConsoleLogger {
	return &ConsoleLogger{
		level:       LevelInfo,
		enableColor: enableColor,
	}
}

func (cl *ConsoleLogger) SetLevel(level LogLevel) {
	cl.mu.Lock()
	cl.level = level
	cl.mu.Unlock()
}

func (cl *ConsoleLogger) formatMessage(level LogLevel, msg string, fields map[string]interface{}) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelStr := cl.levelToString(level)
	if cl.enableColor {
		levelStr = cl.colorize(level, levelStr)
	}

	var fieldsStr string
	if len(fields) > 0 {
		var builder strings.Builder
		builder.WriteString(" | ")
		first := true
		for k, v := range fields {
			if !first {
				builder.WriteString(", ")
			}
			fmt.Fprintf(&builder, "%s=%v", k, v)
			first = false
		}
		fieldsStr = builder.String()
	}

	return fmt.Sprintf("[%s] %s %s%s", timestamp, levelStr, msg, fieldsStr)
}

func (cl *ConsoleLogger) Debug(msg string, fields map[string]interface{}) {
	cl.log(LevelDebug, msg, fields)
}

func (cl *ConsoleLogger) Info(msg string, fields map[string]interface{}) {
	cl.log(LevelInfo, msg, fields)
}

func (cl *ConsoleLogger) Warn(msg string, fields map[string]interface{}) {
	cl.log(LevelWarn, msg, fields)
}

func (cl *ConsoleLogger) Error(msg string, fields map[string]interface{}) {
	cl.log(LevelError, msg, fields)
}

func (cl *ConsoleLogger) Fatal(msg string, fields map[string]interface{}) {
	cl.log(LevelFatal, msg, fields)
}

func (cl *ConsoleLogger) log(level LogLevel, msg string, fields map[string]interface{}) {
	cl.mu.RLock()
	if level < cl.level {
		cl.mu.RUnlock()
		return
	}
	cl.mu.RUnlock()

	fmt.Println(cl.formatMessage(level, msg, fields))
}

func (cl *ConsoleLogger) levelToString(level LogLevel) string {
	switch level {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

func (cl *ConsoleLogger) colorize(level LogLevel, text string) string {
	colors := map[LogLevel]string{
		LevelDebug: "\033[36m",
		LevelInfo:  "\033[32m",
		LevelWarn:  "\033[33m",
		LevelError: "\033[31m",
		LevelFatal: "\033[35m",
	}

	reset := "\033[0m"
	if color, ok := colors[level]; ok {
		return color + text + reset
	}
	return text
}

func (cl *ConsoleLogger) Close() error {
	return nil
}

type FileLogger struct {
	file     *os.File
	level    LogLevel
	filePath string
	mu       sync.Mutex
}

func NewFileLogger(filePath string) (*FileLogger, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &FileLogger{
		file:     file,
		level:    LevelInfo,
		filePath: filePath,
	}, nil
}

func (fl *FileLogger) SetLevel(level LogLevel) {
	fl.mu.Lock()
	fl.level = level
	fl.mu.Unlock()
}

func (fl *FileLogger) Debug(msg string, fields map[string]interface{}) {
	fl.log(LevelDebug, msg, fields)
}

func (fl *FileLogger) Info(msg string, fields map[string]interface{}) {
	fl.log(LevelInfo, msg, fields)
}

func (fl *FileLogger) Warn(msg string, fields map[string]interface{}) {
	fl.log(LevelWarn, msg, fields)
}

func (fl *FileLogger) Error(msg string, fields map[string]interface{}) {
	fl.log(LevelError, msg, fields)
}

func (fl *FileLogger) Fatal(msg string, fields map[string]interface{}) {
	fl.log(LevelFatal, msg, fields)
}

func (fl *FileLogger) log(level LogLevel, msg string, fields map[string]interface{}) {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	if level < fl.level {
		return
	}

	timestamp := time.Now().Format(time.RFC3339)
	levelStr := map[LogLevel]string{
		LevelDebug: "DEBUG",
		LevelInfo:  "INFO",
		LevelWarn:  "WARN",
		LevelError: "ERROR",
		LevelFatal: "FATAL",
	}[level]

	var builder strings.Builder
	fmt.Fprintf(&builder, "[%s] %s %s", timestamp, levelStr, msg)

	if len(fields) > 0 {
		builder.WriteString(" | ")
		first := true
		for k, v := range fields {
			if !first {
				builder.WriteString(", ")
			}
			fmt.Fprintf(&builder, "%s=%v", k, v)
			first = false
		}
	}

	builder.WriteString("\n")
	logLine := builder.String()
	if _, err := fl.file.WriteString(logLine); err != nil {
		fmt.Printf("Failed to write to log file: %v\n", err)
	}
}

func (fl *FileLogger) Close() error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	if fl.file != nil {
		return fl.file.Close()
	}
	return nil
}
