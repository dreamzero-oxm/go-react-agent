package tools

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Tool struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Parameters  map[string]Parameter `json:"parameters"`
	Execute     func(input map[string]interface{}) (string, error)
}

type Parameter struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

var BuiltinTools = []*Tool{
	{
		Name:        "calculate",
		Description: "Perform basic arithmetic calculations. Supports addition, subtraction, multiplication, and division.",
		Parameters: map[string]Parameter{
			"expression": {
				Type:        "string",
				Description: "Mathematical expression to evaluate (e.g., '2 + 3', '10 / 2')",
				Required:    true,
			},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			expr, ok := input["expression"].(string)
			if !ok {
				return "", fmt.Errorf("expression must be a string")
			}

			result, err := EvaluateSimpleMath(expr)
			if err != nil {
				return "", fmt.Errorf("calculation error: %w", err)
			}

			return fmt.Sprintf("Result: %s = %g", expr, result), nil
		},
	},
	{
		Name:        "http_get",
		Description: "Make an HTTP GET request to a URL and return response",
		Parameters: map[string]Parameter{
			"url": {
				Type:        "string",
				Description: "The URL to make the request to",
				Required:    true,
			},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			url, ok := input["url"].(string)
			if !ok {
				return "", fmt.Errorf("url must be a string")
			}

			resp, err := http.Get(url)
			if err != nil {
				return "", fmt.Errorf("HTTP request failed: %w", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", fmt.Errorf("failed to read response body: %w", err)
			}

			return fmt.Sprintf("Status: %d\nBody: %s", resp.StatusCode, string(body)), nil
		},
	},
	{
		Name:        "read_file",
		Description: "Read the contents of a file",
		Parameters: map[string]Parameter{
			"path": {
				Type:        "string",
				Description: "Path to the file to read",
				Required:    true,
			},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			path, ok := input["path"].(string)
			if !ok {
				return "", fmt.Errorf("path must be a string")
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return "", fmt.Errorf("failed to read file: %w", err)
			}

			return string(content), nil
		},
	},
	{
		Name:        "write_file",
		Description: "Write content to a file",
		Parameters: map[string]Parameter{
			"path": {
				Type:        "string",
				Description: "Path to the file to write",
				Required:    true,
			},
			"content": {
				Type:        "string",
				Description: "Content to write to the file",
				Required:    true,
			},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			path, ok := input["path"].(string)
			if !ok {
				return "", fmt.Errorf("path must be a string")
			}

			content, ok := input["content"].(string)
			if !ok {
				return "", fmt.Errorf("content must be a string")
			}

			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return "", fmt.Errorf("failed to write file: %w", err)
			}

			return fmt.Sprintf("Successfully wrote to %s", path), nil
		},
	},
	{
		Name:        "echo",
		Description: "Echo back the provided text",
		Parameters: map[string]Parameter{
			"text": {
				Type:        "string",
				Description: "Text to echo back",
				Required:    true,
			},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			text, ok := input["text"].(string)
			if !ok {
				return "", fmt.Errorf("text must be a string")
			}
			return text, nil
		},
	},
	{
		Name:        "search_files",
		Description: "Search for files in a directory matching a pattern",
		Parameters: map[string]Parameter{
			"directory": {
				Type:        "string",
				Description: "Directory to search in",
				Required:    true,
			},
			"pattern": {
				Type:        "string",
				Description: "File pattern to match (e.g., '*.txt', 'test_*.go')",
				Required:    true,
			},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			directory, ok := input["directory"].(string)
			if !ok {
				return "", fmt.Errorf("directory must be a string")
			}

			pattern, ok := input["pattern"].(string)
			if !ok {
				return "", fmt.Errorf("pattern must be a string")
			}

			matches, err := FindFiles(directory, pattern)
			if err != nil {
				return "", fmt.Errorf("search failed: %w", err)
			}

			if len(matches) == 0 {
				return "No files found matching the pattern", nil
			}

			return fmt.Sprintf("Found %d files:\n%s", len(matches), strings.Join(matches, "\n")), nil
		},
	},
}

func RegisterBuiltinTools(registry *ToolRegistry) {
	for _, tool := range BuiltinTools {
		registry.RegisterTool(tool)
	}
}

type AgentToolRegistry interface {
	RegisterTool(tool interface{}) error
	UnregisterTool(name string) error
	Get(name string) (interface{}, error)
	List() []string
	Execute(name string, input map[string]interface{}) (string, error)
}

func RegisterBuiltinToolsTo(registry AgentToolRegistry) {
	for _, tool := range BuiltinTools {
		registry.RegisterTool(tool)
	}
}

func EvaluateSimpleMath(expr string) (float64, error) {
	expr = strings.TrimSpace(expr)
	expr = strings.ReplaceAll(expr, " ", "")

	var parseExpression func(s string) (float64, int)
	var parseTerm func(s string) (float64, int)
	var parseFactor func(s string) (float64, int)

	parseFactor = func(s string) (float64, int) {
		if len(s) == 0 {
			return 0, 0
		}

		if s[0] == '(' {
			val, pos := parseExpression(s[1:])
			if pos >= len(s) || s[pos] != ')' {
				return 0, 0
			}
			return val, pos + 2
		}

		if s[0] == '-' {
			val, pos := parseFactor(s[1:])
			return -val, pos + 1
		}

		numStr := ""
		for i := 0; i < len(s); i++ {
			c := s[i]
			if c >= '0' && c <= '9' || c == '.' {
				numStr += string(c)
			} else {
				break
			}
		}

		if numStr == "" {
			return 0, 0
		}

		val, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return 0, 0
		}

		return val, len(numStr)
	}

	parseTerm = func(s string) (float64, int) {
		left, pos := parseFactor(s)

		for pos < len(s) && (s[pos] == '*' || s[pos] == '/') {
			op := s[pos]
			right, newPos := parseFactor(s[pos+1:])

			if op == '*' {
				left *= right
			} else {
				if right == 0 {
					return 0, 0
				}
				left /= right
			}
			pos = pos + 1 + newPos
		}

		return left, pos
	}

	parseExpression = func(s string) (float64, int) {
		left, pos := parseTerm(s)

		for pos < len(s) && (s[pos] == '+' || s[pos] == '-') {
			op := s[pos]
			right, newPos := parseTerm(s[pos+1:])

			if op == '+' {
				left += right
			} else {
				left -= right
			}
			pos = pos + 1 + newPos
		}

		return left, pos
	}

	result, pos := parseExpression(expr)
	if pos != len(expr) {
		return 0, fmt.Errorf("invalid expression: %s", expr)
	}

	return result, nil
}

func FindFiles(dir, pattern string) ([]string, error) {
	var matches []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matched, err := matchPattern(entry.Name(), pattern)
		if err != nil {
			continue
		}

		if matched {
			matches = append(matches, entry.Name())
		}
	}

	return matches, nil
}

func matchPattern(name, pattern string) (bool, error) {
	pattern = strings.ReplaceAll(pattern, ".", "\\.")
	pattern = strings.ReplaceAll(pattern, "*", ".*")

	return name == pattern || strings.Contains(name, strings.ReplaceAll(pattern, ".*", "")), nil
}
