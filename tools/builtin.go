package tools

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
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
		Description: "Perform basic arithmetic calculations. Supports addition (+), subtraction (-), multiplication (*), division (/), and parentheses. Example: '2 + 3 * 4', '(10 + 5) / 3'",
		Parameters: map[string]Parameter{
			"expression": {
				Type:        "string",
				Description: "Mathematical expression to evaluate. Supports: +, -, *, /, (, ). Example: '2 + 3', '10 / 2', '(1 + 2) * 3'",
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
		Description: "Make an HTTP GET request to a URL and return response. Returns status code and response body.",
		Parameters: map[string]Parameter{
			"url": {
				Type:        "string",
				Description: "The full URL to make the HTTP GET request to (e.g., 'https://api.example.com/data')",
				Required:    true,
			},
			"headers": {
				Type:        "object",
				Description: "Optional HTTP headers to include in the request (e.g., {'Authorization': 'Bearer token', 'Content-Type': 'application/json'})",
				Required:    false,
			},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			urlStr, ok := input["url"].(string)
			if !ok {
				return "", fmt.Errorf("url must be a string")
			}

			req, err := http.NewRequest("GET", urlStr, nil)
			if err != nil {
				return "", fmt.Errorf("failed to create request: %w", err)
			}

			if headers, ok := input["headers"].(map[string]interface{}); ok {
				for key, value := range headers {
					if strValue, ok := value.(string); ok {
						req.Header.Set(key, strValue)
					}
				}
			}

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				return "", fmt.Errorf("HTTP request failed: %w", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return "", fmt.Errorf("failed to read response body: %w", err)
			}

			return fmt.Sprintf("Status: %d %s\nBody: %s", resp.StatusCode, resp.Status, string(body)), nil
		},
	},
	{
		Name:        "read_file",
		Description: "Read and return the complete contents of a file at the specified path.",
		Parameters: map[string]Parameter{
			"path": {
				Type:        "string",
				Description: "Full or relative path to the file to read (e.g., './config.json', '/home/user/data.txt')",
				Required:    true,
			},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			path, ok := input["path"].(string)
			if !ok {
				return "", fmt.Errorf("path must be a string")
			}

			absPath, err := filepath.Abs(path)
			if err != nil {
				return "", fmt.Errorf("failed to resolve absolute path: %w", err)
			}

			content, err := os.ReadFile(absPath)
			if err != nil {
				return "", fmt.Errorf("failed to read file '%s': %w", absPath, err)
			}

			return string(content), nil
		},
	},
	{
		Name:        "write_file",
		Description: "Write content to a file. Creates the file if it doesn't exist, overwrites if it does.",
		Parameters: map[string]Parameter{
			"path": {
				Type:        "string",
				Description: "Full or relative path to the file to write (e.g., './output.txt', '/tmp/data.json')",
				Required:    true,
			},
			"content": {
				Type:        "string",
				Description: "The text content to write to the file",
				Required:    true,
			},
			"append": {
				Type:        "boolean",
				Description: "If true, append to existing file instead of overwriting. Default: false",
				Required:    false,
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

			appendMode, _ := input["append"].(bool)

			absPath, err := filepath.Abs(path)
			if err != nil {
				return "", fmt.Errorf("failed to resolve absolute path: %w", err)
			}

			var flags int
			if appendMode {
				flags = os.O_APPEND | os.O_CREATE | os.O_WRONLY
			} else {
				flags = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
			}

			file, err := os.OpenFile(absPath, flags, 0644)
			if err != nil {
				return "", fmt.Errorf("failed to open file '%s': %w", absPath, err)
			}
			defer file.Close()

			if _, err := file.WriteString(content); err != nil {
				return "", fmt.Errorf("failed to write to file '%s': %w", absPath, err)
			}

			action := "wrote to"
			if appendMode {
				action = "appended to"
			}
			return fmt.Sprintf("Successfully %s %s (%d bytes)", action, absPath, len(content)), nil
		},
	},
	{
		Name:        "delete_file",
		Description: "Delete a file at the specified path. Use with caution - this operation cannot be undone.",
		Parameters: map[string]Parameter{
			"path": {
				Type:        "string",
				Description: "Full or relative path to the file to delete (e.g., './temp.txt', '/tmp/old_data.json')",
				Required:    true,
			},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			path, ok := input["path"].(string)
			if !ok {
				return "", fmt.Errorf("path must be a string")
			}

			absPath, err := filepath.Abs(path)
			if err != nil {
				return "", fmt.Errorf("failed to resolve absolute path: %w", err)
			}

			if err := os.Remove(absPath); err != nil {
				return "", fmt.Errorf("failed to delete file '%s': %w", absPath, err)
			}

			return fmt.Sprintf("Successfully deleted file: %s", absPath), nil
		},
	},
	{
		Name:        "list_files",
		Description: "List all files and directories in a specified directory, with optional filtering by extension or pattern.",
		Parameters: map[string]Parameter{
			"directory": {
				Type:        "string",
				Description: "Directory path to list contents from (e.g., '.', './src', '/home/user')",
				Required:    true,
			},
			"include_hidden": {
				Type:        "boolean",
				Description: "If true, include hidden files (starting with .). Default: false",
				Required:    false,
			},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			directory, ok := input["directory"].(string)
			if !ok {
				return "", fmt.Errorf("directory must be a string")
			}

			includeHidden, _ := input["include_hidden"].(bool)

			absPath, err := filepath.Abs(directory)
			if err != nil {
				return "", fmt.Errorf("failed to resolve absolute path: %w", err)
			}

			entries, err := os.ReadDir(absPath)
			if err != nil {
				return "", fmt.Errorf("failed to read directory '%s': %w", absPath, err)
			}

			var result []string
			for _, entry := range entries {
				if !includeHidden && strings.HasPrefix(entry.Name(), ".") {
					continue
				}

				fileType := "file"
				if entry.IsDir() {
					fileType = "dir"
				}

				size := ""
				if !entry.IsDir() {
					info, err := entry.Info()
					if err == nil {
						size = fmt.Sprintf(" (%d bytes)", info.Size())
					}
				}

				result = append(result, fmt.Sprintf("[%s] %s%s", fileType, entry.Name(), size))
			}

			if len(result) == 0 {
				return fmt.Sprintf("Directory is empty: %s", absPath), nil
			}

			return fmt.Sprintf("Directory: %s\n\n%s", absPath, strings.Join(result, "\n")), nil
		},
	},
	{
		Name:        "create_directory",
		Description: "Create a new directory. Creates parent directories if they don't exist (like mkdir -p).",
		Parameters: map[string]Parameter{
			"path": {
				Type:        "string",
				Description: "Path of the directory to create (e.g., './data/exports', '/tmp/project/logs')",
				Required:    true,
			},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			path, ok := input["path"].(string)
			if !ok {
				return "", fmt.Errorf("path must be a string")
			}

			absPath, err := filepath.Abs(path)
			if err != nil {
				return "", fmt.Errorf("failed to resolve absolute path: %w", err)
			}

			if err := os.MkdirAll(absPath, 0755); err != nil {
				return "", fmt.Errorf("failed to create directory '%s': %w", absPath, err)
			}

			return fmt.Sprintf("Successfully created directory: %s", absPath), nil
		},
	},
	{
		Name:        "echo",
		Description: "Echo back provided text. Useful for debugging or returning formatted output.",
		Parameters: map[string]Parameter{
			"text": {
				Type:        "string",
				Description: "The text to echo back",
				Required:    true,
			},
			"uppercase": {
				Type:        "boolean",
				Description: "If true, convert text to uppercase. Default: false",
				Required:    false,
			},
			"lowercase": {
				Type:        "boolean",
				Description: "If true, convert text to lowercase. Default: false",
				Required:    false,
			},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			text, ok := input["text"].(string)
			if !ok {
				return "", fmt.Errorf("text must be a string")
			}

			result := text
			if uppercase, ok := input["uppercase"].(bool); ok && uppercase {
				result = strings.ToUpper(result)
			}
			if lowercase, ok := input["lowercase"].(bool); ok && lowercase {
				result = strings.ToLower(result)
			}

			return result, nil
		},
	},
	{
		Name:        "search_files",
		Description: "Search for files in a directory matching a pattern. Supports wildcard patterns like *.txt or test_*.go.",
		Parameters: map[string]Parameter{
			"directory": {
				Type:        "string",
				Description: "Directory to search in (e.g., '.', './src', '/tmp')",
				Required:    true,
			},
			"pattern": {
				Type:        "string",
				Description: "File pattern to match. Supports * wildcard (e.g., '*.txt', 'test_*.go', 'config.*')",
				Required:    true,
			},
			"recursive": {
				Type:        "boolean",
				Description: "If true, search subdirectories recursively. Default: false",
				Required:    false,
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

			recursive, _ := input["recursive"].(bool)

			absPath, err := filepath.Abs(directory)
			if err != nil {
				return "", fmt.Errorf("failed to resolve absolute path: %w", err)
			}

			var matches []string
			if recursive {
				matches, err = FindFilesRecursive(absPath, pattern)
			} else {
				matches, err = FindFiles(absPath, pattern)
			}

			if err != nil {
				return "", fmt.Errorf("search failed: %w", err)
			}

			if len(matches) == 0 {
				return fmt.Sprintf("No files found matching pattern '%s' in directory '%s'", pattern, absPath), nil
			}

			return fmt.Sprintf("Found %d file(s) matching '%s' in '%s':\n%s", len(matches), pattern, absPath, strings.Join(matches, "\n")), nil
		},
	},
	{
		Name:        "current_time",
		Description: "Get the current date and time in various formats.",
		Parameters: map[string]Parameter{
			"format": {
				Type:        "string",
				Description: "Optional time format. Common formats: 'rfc3339' (default), 'unix', 'iso8601', 'date', 'time'",
				Required:    false,
			},
			"timezone": {
				Type:        "string",
				Description: "Optional timezone (e.g., 'UTC', 'America/New_York', 'Asia/Shanghai'). Default: local timezone",
				Required:    false,
			},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			formatStr, _ := input["format"].(string)
			timezoneStr, _ := input["timezone"].(string)

			now := time.Now()

			if timezoneStr != "" {
				loc, err := time.LoadLocation(timezoneStr)
				if err != nil {
					return "", fmt.Errorf("invalid timezone '%s': %w", timezoneStr, err)
				}
				now = now.In(loc)
			}

			var result string
			switch strings.ToLower(formatStr) {
			case "rfc3339":
				result = now.Format(time.RFC3339)
			case "unix":
				result = fmt.Sprintf("%d", now.Unix())
			case "iso8601":
				result = now.Format(time.RFC3339)
			case "date":
				result = now.Format("2006-01-02")
			case "time":
				result = now.Format("15:04:05")
			case "datetime":
				result = now.Format("2006-01-02 15:04:05")
			default:
				result = fmt.Sprintf("RFC3339: %s\nISO8601: %s\nUnix: %d\nDate: %s\nTime: %s",
					now.Format(time.RFC3339),
					now.Format(time.RFC3339),
					now.Unix(),
					now.Format("2006-01-02"),
					now.Format("15:04:05"),
				)
			}

			return result, nil
		},
	},
	{
		Name:        "format_text",
		Description: "Format text using various transformations and templates.",
		Parameters: map[string]Parameter{
			"text": {
				Type:        "string",
				Description: "The text to format",
				Required:    true,
			},
			"operation": {
				Type:        "string",
				Description: "Operation to perform: 'uppercase', 'lowercase', 'title', 'reverse', 'trim', 'replace'",
				Required:    false,
			},
			"replace_old": {
				Type:        "string",
				Description: "Text to replace (used with operation='replace')",
				Required:    false,
			},
			"replace_new": {
				Type:        "string",
				Description: "Replacement text (used with operation='replace')",
				Required:    false,
			},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			text, ok := input["text"].(string)
			if !ok {
				return "", fmt.Errorf("text must be a string")
			}

			operation, _ := input["operation"].(string)

			var result string
			switch strings.ToLower(operation) {
			case "uppercase":
				result = strings.ToUpper(text)
			case "lowercase":
				result = strings.ToLower(text)
			case "title":
				result = strings.ToTitle(text)
			case "reverse":
				runes := []rune(text)
				for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
					runes[i], runes[j] = runes[j], runes[i]
				}
				result = string(runes)
			case "trim":
				result = strings.TrimSpace(text)
			case "replace":
				oldStr, _ := input["replace_old"].(string)
				newStr, _ := input["replace_new"].(string)
				if oldStr == "" {
					return "", fmt.Errorf("replace_old is required when operation='replace'")
				}
				result = strings.ReplaceAll(text, oldStr, newStr)
			default:
				result = text
			}

			return result, nil
		},
	},
	{
		Name:        "base64_encode",
		Description: "Encode text or data to Base64 format.",
		Parameters: map[string]Parameter{
			"data": {
				Type:        "string",
				Description: "Text or data to encode to Base64",
				Required:    true,
			},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			data, ok := input["data"].(string)
			if !ok {
				return "", fmt.Errorf("data must be a string")
			}

			encoded := base64.StdEncoding.EncodeToString([]byte(data))
			return encoded, nil
		},
	},
	{
		Name:        "base64_decode",
		Description: "Decode Base64 encoded data back to text.",
		Parameters: map[string]Parameter{
			"data": {
				Type:        "string",
				Description: "Base64 encoded data to decode",
				Required:    true,
			},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			data, ok := input["data"].(string)
			if !ok {
				return "", fmt.Errorf("data must be a string")
			}

			decoded, err := base64.StdEncoding.DecodeString(data)
			if err != nil {
				return "", fmt.Errorf("failed to decode Base64: %w", err)
			}

			return string(decoded), nil
		},
	},
	{
		Name:        "regex_match",
		Description: "Test if text matches a regular expression pattern and optionally extract groups.",
		Parameters: map[string]Parameter{
			"pattern": {
				Type:        "string",
				Description: "Regular expression pattern to match (Go regex syntax)",
				Required:    true,
			},
			"text": {
				Type:        "string",
				Description: "Text to test against the pattern",
				Required:    true,
			},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			patternStr, ok := input["pattern"].(string)
			if !ok {
				return "", fmt.Errorf("pattern must be a string")
			}

			text, ok := input["text"].(string)
			if !ok {
				return "", fmt.Errorf("text must be a string")
			}

			matched := regexp.MustCompile(patternStr).MatchString(text)
			if matched {
				return fmt.Sprintf("Pattern '%s' matched text", patternStr), nil
			}
			return fmt.Sprintf("Pattern '%s' did NOT match text", patternStr), nil
		},
	},
	{
		Name:        "json_parse",
		Description: "Parse and validate JSON string, optionally extract a specific field value.",
		Parameters: map[string]Parameter{
			"json_string": {
				Type:        "string",
				Description: "JSON string to parse",
				Required:    true,
			},
			"field": {
				Type:        "string",
				Description: "Optional field path to extract (e.g., 'data.items[0].name')",
				Required:    false,
			},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			jsonStr, ok := input["json_string"].(string)
			if !ok {
				return "", fmt.Errorf("json_string must be a string")
			}

			var data interface{}
			if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
				return "", fmt.Errorf("failed to parse JSON: %w", err)
			}

			fieldPath, hasField := input["field"].(string)
			if !hasField {
				formatted, _ := json.MarshalIndent(data, "", "  ")
				return string(formatted), nil
			}

			value, err := extractJSONField(data, strings.Split(fieldPath, "."))
			if err != nil {
				return "", fmt.Errorf("failed to extract field '%s': %w", fieldPath, err)
			}

			formatted, _ := json.MarshalIndent(value, "", "  ")
			return string(formatted), nil
		},
	},
	{
		Name:        "url_encode",
		Description: "URL encode a string for safe use in query parameters.",
		Parameters: map[string]Parameter{
			"text": {
				Type:        "string",
				Description: "Text to URL encode",
				Required:    true,
			},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			text, ok := input["text"].(string)
			if !ok {
				return "", fmt.Errorf("text must be a string")
			}

			encoded := url.QueryEscape(text)
			return encoded, nil
		},
	},
	{
		Name:        "url_decode",
		Description: "Decode a URL encoded string back to plain text.",
		Parameters: map[string]Parameter{
			"text": {
				Type:        "string",
				Description: "URL encoded text to decode",
				Required:    true,
			},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			text, ok := input["text"].(string)
			if !ok {
				return "", fmt.Errorf("text must be a string")
			}

			decoded, err := url.QueryUnescape(text)
			if err != nil {
				return "", fmt.Errorf("failed to decode URL: %w", err)
			}

			return decoded, nil
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

func RegisterBuiltinToolsTo(registry AgentToolRegistry) error {
	for _, tool := range BuiltinTools {
		err := registry.RegisterTool(tool)
		if err != nil {
			return err
		}
	}
	return nil
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

func FindFilesRecursive(dir, pattern string) ([]string, error) {
	var matches []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		relPath := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			subMatches, err := FindFilesRecursive(relPath, pattern)
			if err != nil {
				continue
			}
			matches = append(matches, subMatches...)
		} else {
			matched, err := matchPattern(entry.Name(), pattern)
			if err != nil {
				continue
			}
			if matched {
				matches = append(matches, relPath)
			}
		}
	}

	return matches, nil
}

func matchPattern(name, pattern string) (bool, error) {
	pattern = strings.ReplaceAll(pattern, ".", "\\.")
	pattern = strings.ReplaceAll(pattern, "*", ".*")
	return name == pattern || strings.Contains(name, strings.ReplaceAll(pattern, ".*", "")), nil
}

func extractJSONField(data interface{}, path []string) (interface{}, error) {
	current := data

	for i, part := range path {
		switch v := current.(type) {
		case map[string]interface{}:
			if val, exists := v[part]; exists {
				current = val
			} else {
				return nil, fmt.Errorf("field not found: %s", strings.Join(path[:i+1], "."))
			}
		case []interface{}:
			index, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid array index: %s", part)
			}
			if index < 0 || index >= len(v) {
				return nil, fmt.Errorf("array index out of bounds: %s", part)
			}
			current = v[index]
		default:
			return nil, fmt.Errorf("cannot access field at path: %s", strings.Join(path[:i+1], "."))
		}
	}

	return current, nil
}
