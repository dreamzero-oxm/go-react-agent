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

func NewCalculateTool() *Tool {
	return &Tool{
		Name:        "calculate",
		Description: "Evaluates basic arithmetic expressions and returns the calculated result. Supports addition (+), subtraction (-), multiplication (*), division (/), and parentheses for grouping operations. The calculator follows standard mathematical operator precedence where multiplication and division are performed before addition and subtraction. Parentheses can be used to override the default precedence and ensure operations are performed in the desired order. This tool is useful for performing mathematical calculations, verifying arithmetic results, or computing numerical values based on user-provided expressions. The tool handles floating-point numbers and will return an error if the expression is invalid or if division by zero is attempted.",
		Parameters: map[string]Parameter{
			"expression": {
				Type:        "string",
				Description: "The mathematical expression to evaluate. Supports the following operators and symbols: addition (+), subtraction (-), multiplication (*), division (/), left parenthesis ((), and right parenthesis ()). Numbers can be integers or decimal values. Whitespace in the expression is automatically ignored. Examples: '2 + 3' evaluates to 5, '10 / 2' evaluates to 5, '(1 + 2) * 3' evaluates to 9, '2 + 3 * 4' evaluates to 14 (multiplication performed first), '(10 + 5) / 3' evaluates to 5",
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
	}
}

func NewHttpGetTool() *Tool {
	return &Tool{
		Name:        "http_get",
		Description: "Performs an HTTP GET request to the specified URL and returns the complete HTTP response including the status code, status message, and response body. This tool enables agents to interact with web services, retrieve data from APIs, fetch web pages, or communicate with external systems. The request includes a 30-second timeout to prevent hanging on unresponsive servers. Custom HTTP headers can be included in the request for authentication, content negotiation, or other HTTP protocol requirements. The tool automatically handles the HTTP connection, follows redirects (if any), and returns the raw response body as a string, which may contain JSON, XML, HTML, or plain text depending on the API or web service being accessed.",
		Parameters: map[string]Parameter{
			"url": {
				Type:        "string",
				Description: "The complete URL to which the HTTP GET request will be sent. The URL must include the protocol (http:// or https://), domain name, and any path or query parameters. Examples: 'https://api.example.com/data', 'https://jsonplaceholder.typicode.com/posts/1', 'http://localhost:8080/api/status'. The URL will be used exactly as provided without modification.",
				Required:    true,
			},
			"headers": {
				Type:        "object",
				Description: "Optional HTTP headers to include in the request as key-value pairs. Headers are commonly used for authentication (e.g., Authorization header), specifying content types (e.g., Content-Type header), or providing additional metadata to the server. Each key-value pair represents a header name and its corresponding value. Example: {'Authorization': 'Bearer token123', 'Content-Type': 'application/json', 'Accept': 'application/json'}. If not provided, only default HTTP headers will be sent.",
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
	}
}

func NewReadFileTool() *Tool {
	return &Tool{
		Name:        "read_file",
		Description: "Reads the complete contents of a file at the specified path and returns the text as a string. This tool supports both relative and absolute paths, automatically resolving relative paths to absolute paths before reading. The tool is useful for accessing configuration files, reading data files, retrieving source code, or processing any text-based file content. The entire file is read into memory, so for very large files (hundreds of megabytes or more), consider using alternative approaches. Binary files will be read, but their content may not be meaningfully represented as text. The tool returns an error if the file does not exist, if there are insufficient permissions to read the file, or if the path refers to a directory rather than a file.",
		Parameters: map[string]Parameter{
			"path": {
				Type:        "string",
				Description: "The path to the file to be read. Both relative and absolute paths are supported. Relative paths are resolved relative to the current working directory of the process. Examples of valid paths: './config.json' (file in current directory), '../data.txt' (file in parent directory), '/home/user/data.txt' (absolute path on Unix systems), 'C:\\Users\\user\\file.txt' (absolute path on Windows). The path separator can be either forward slash (/) or backslash (\\) depending on the operating system.",
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
	}
}

func NewWriteFileTool() *Tool {
	return &Tool{
		Name:        "write_file",
		Description: "Writes text content to a file at the specified path. If the file does not exist, it will be created. If the file already exists, it can either be overwritten (default behavior) or appended to based on the 'append' parameter. The tool supports both relative and absolute paths, automatically resolving relative paths to absolute paths. File permissions are set to 0644 (read/write for owner, read-only for group and others) which is a standard permission level for text files. This tool is useful for creating log files, saving configuration data, writing output from computations, or persisting any text-based information to disk. The tool returns an error if the parent directory does not exist, if there are insufficient permissions to write to the location, or if the disk is full.",
		Parameters: map[string]Parameter{
			"path": {
				Type:        "string",
				Description: "The path where the file should be written. Both relative and absolute paths are supported. Relative paths are resolved relative to the current working directory. The path can include directory separators to create files in subdirectories, but the parent directory must already exist. Examples: './output.txt' (create file in current directory), '/tmp/data.json' (create file in system temp directory), './logs/app.log' (create file in logs subdirectory).",
				Required:    true,
			},
			"content": {
				Type:        "string",
				Description: "The text content to write to the file. This can be any UTF-8 text including newlines, special characters, and Unicode symbols. The content is written exactly as provided without any automatic formatting or encoding changes. Empty strings will result in an empty file. For binary data, consider encoding it (e.g., Base64) before passing as a string parameter.",
				Required:    true,
			},
			"append": {
				Type:        "boolean",
				Description: "Determines whether to append content to an existing file or overwrite it. When set to true, new content is added to the end of the file without modifying existing content. When set to false (default), the entire file is replaced with the new content. If the file does not exist, this parameter has no effect and the file is created. Default value: false",
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
	}
}

func NewDeleteFileTool() *Tool {
	return &Tool{
		Name:        "delete_file",
		Description: "Permanently deletes the file at the specified path. This operation cannot be undone, so caution should be exercised when using this tool. The tool supports both relative and absolute paths, automatically resolving relative paths to absolute paths before deletion. Only files can be deleted with this tool; directories must be removed using directory-specific operations. The tool is useful for cleaning up temporary files, removing outdated data, or managing file lifecycle operations. The tool returns an error if the file does not exist, if there are insufficient permissions to delete the file, if the path refers to a directory, or if the file is currently in use by another process (on some operating systems).",
		Parameters: map[string]Parameter{
			"path": {
				Type:        "string",
				Description: "The path to the file to be deleted. Both relative and absolute paths are supported. Relative paths are resolved relative to the current working directory. The path must point to a regular file, not a directory. Examples: './temp.txt' (delete file in current directory), '/tmp/old_data.json' (delete file in system temp directory), './cache/session.cache' (delete file in cache subdirectory). Use with caution as deletion is permanent.",
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
	}
}

func NewListFilesTool() *Tool {
	return &Tool{
		Name:        "list_files",
		Description: "Lists all files and subdirectories contained within the specified directory. The tool provides detailed information about each entry including whether it is a file or directory and, for files, the size in bytes. Hidden files (those starting with a dot on Unix systems) can optionally be included or excluded based on the 'include_hidden' parameter. The tool supports both relative and absolute paths, automatically resolving relative paths to absolute paths. This tool is useful for exploring directory structures, finding files, checking directory contents, or analyzing file system organization. The listing is not recursive; only the immediate contents of the specified directory are returned. The tool returns an error if the directory does not exist, if there are insufficient permissions to read the directory, or if the path refers to a file rather than a directory.",
		Parameters: map[string]Parameter{
			"directory": {
				Type:        "string",
				Description: "The path to the directory whose contents should be listed. Both relative and absolute paths are supported. Relative paths are resolved relative to the current working directory. The path must point to a valid directory. Examples: '.' (list current directory), './src' (list src subdirectory), '/home/user' (list user home directory), './project/data' (list data subdirectory). Special characters and spaces in directory names are supported.",
				Required:    true,
			},
			"include_hidden": {
				Type:        "boolean",
				Description: "Controls whether hidden files and directories should be included in the listing. On Unix systems, hidden files are those whose names start with a dot (e.g., '.gitignore', '.hidden_file'). On Windows, files with the hidden attribute set are considered hidden. When set to true, all files including hidden ones are returned. When set to false (default), hidden files are excluded from the listing. Default value: false",
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
	}
}

func NewCreateDirectoryTool() *Tool {
	return &Tool{
		Name:        "create_directory",
		Description: "Creates a new directory at the specified path. If parent directories in the path do not exist, they are automatically created, similar to the 'mkdir -p' command in Unix systems. The tool supports both relative and absolute paths, automatically resolving relative paths to absolute paths. Directory permissions are set to 0755 (read, write, execute for owner; read and execute for group and others) which is a standard permission level for directories. This tool is useful for creating project directories, organizing file structures, preparing storage locations, or establishing directory hierarchies. The tool returns an error if there are insufficient permissions to create the directory, if the path already exists as a file, or if the disk is full. If the directory already exists, no error is returned.",
		Parameters: map[string]Parameter{
			"path": {
				Type:        "string",
				Description: "The path of the directory to create. Both relative and absolute paths are supported. Relative paths are resolved relative to the current working directory. The path can include multiple directory levels separated by path separators; any missing intermediate directories will be created automatically. Examples: './data/exports' (create nested directories), '/tmp/project/logs' (create absolute path), './cache/temp' (create multiple levels at once). Special characters and spaces in directory names are supported.",
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
	}
}

func NewEchoTool() *Tool {
	return &Tool{
		Name:        "echo",
		Description: "Returns the provided text content unchanged, with optional case transformation. This tool is primarily useful for debugging purposes, testing agent behavior, or formatting and returning output in a specific case. The tool can convert text to uppercase, lowercase, or leave it unchanged. When both uppercase and lowercase parameters are set to true, lowercase takes precedence. This tool does not perform any text analysis, parsing, or modification beyond case conversion. It is particularly helpful when you need to return a specific string to the user or when testing how the agent processes tool outputs.",
		Parameters: map[string]Parameter{
			"text": {
				Type:        "string",
				Description: "The text content to be returned. This can be any UTF-8 text including letters, numbers, symbols, spaces, newlines, and Unicode characters. The text is returned as provided, subject only to optional case transformation. Examples: 'Hello World', 'This is a test message', '12345', '!@#$%^&*()'. Empty strings are also valid.",
				Required:    true,
			},
			"uppercase": {
				Type:        "boolean",
				Description: "When set to true, converts the entire text to uppercase letters. All lowercase letters are transformed to their uppercase equivalents. Non-alphabetic characters remain unchanged. This is useful for formatting output or creating attention-grabbing text. If both uppercase and lowercase are set to true, lowercase takes precedence. Default value: false",
				Required:    false,
			},
			"lowercase": {
				Type:        "boolean",
				Description: "When set to true, converts the entire text to lowercase letters. All uppercase letters are transformed to their lowercase equivalents. Non-alphabetic characters remain unchanged. This is useful for normalizing text or creating uniform output. If both uppercase and lowercase are set to true, lowercase takes precedence over uppercase. Default value: false",
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
	}
}

func NewSearchFilesTool() *Tool {
	return &Tool{
		Name:        "search_files",
		Description: "Searches for files within a specified directory that match a given pattern. The pattern supports wildcard matching using the asterisk (*) character, which matches zero or more characters. The search can be limited to the immediate directory or extended recursively through all subdirectories based on the 'recursive' parameter. Both relative and absolute paths are supported for the directory parameter, automatically resolving relative paths to absolute paths. This tool is useful for finding files by name pattern, locating specific file types, discovering files in a project structure, or auditing directory contents. The matching is case-sensitive and applies only to file names, not to directory names in the path. The tool returns an error if the directory does not exist or if there are insufficient permissions to read it.",
		Parameters: map[string]Parameter{
			"directory": {
				Type:        "string",
				Description: "The directory to search within for matching files. Both relative and absolute paths are supported. Relative paths are resolved relative to the current working directory. The directory must exist and be accessible. Examples: '.' (search current directory), './src' (search src subdirectory only), '/tmp' (search system temp directory), './project' (search project directory and optionally its subdirectories if recursive is true).",
				Required:    true,
			},
			"pattern": {
				Type:        "string",
				Description: "The file matching pattern to use when searching. The pattern supports the asterisk (*) wildcard which matches zero or more characters. Matching is case-sensitive and applies only to the file name (not the full path or directory name). Common patterns: '*.txt' (all files ending with .txt), 'test_*.go' (all Go files starting with test_), 'config.*' (all files named config with any extension), '*.json' (all JSON files), '*_backup' (all files ending with _backup). The pattern can also match exact file names if no wildcards are used.",
				Required:    true,
			},
			"recursive": {
				Type:        "boolean",
				Description: "Controls whether the search should extend recursively into subdirectories. When set to true, the tool searches the specified directory and all nested subdirectories, returning full relative paths from the search directory to each matching file. When set to false (default), only files in the immediate directory are searched. Recursive searches may take longer for deep directory structures. Default value: false",
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
	}
}

func NewCurrentTimeTool() *Tool {
	return &Tool{
		Name:        "current_time",
		Description: "Retrieves the current date and time from the system clock. The tool supports multiple output formats and optional timezone specification, allowing flexible time representation for different use cases. When no format or timezone is specified, the tool returns a comprehensive display showing multiple common time formats simultaneously. This tool is useful for timestamping events, logging current time, displaying time to users in a preferred format, or performing time-based calculations. The tool can return time in various standard formats including RFC3339, Unix timestamp, ISO8601, simple date, simple time, and combined date-time formats. Timezone support allows conversion to any valid IANA timezone identifier.",
		Parameters: map[string]Parameter{
			"format": {
				Type:        "string",
				Description: "Specifies the desired output format for the time. Supported formats include: 'rfc3339' (RFC 3339 format, e.g., '2024-02-20T15:30:45Z'), 'unix' (Unix timestamp in seconds since epoch, e.g., '1708426245'), 'iso8601' (ISO 8601 format, same as RFC3339), 'date' (date only, e.g., '2024-02-20'), 'time' (time only, e.g., '15:30:45'), 'datetime' (date and time, e.g., '2024-02-20 15:30:45'). If not specified, the tool returns a comprehensive display with all formats. Format matching is case-insensitive.",
				Required:    false,
			},
			"timezone": {
				Type:        "string",
				Description: "Optional timezone specification using IANA timezone identifiers. When provided, the time is converted to the specified timezone. If not provided, the local system timezone is used. Common timezone examples: 'UTC' (Coordinated Universal Time), 'America/New_York' (Eastern Time), 'America/Los_Angeles' (Pacific Time), 'Europe/London' (Greenwich Mean Time), 'Asia/Shanghai' (China Standard Time), 'Asia/Tokyo' (Japan Standard Time). Invalid timezone identifiers will result in an error. Default value: local system timezone",
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
	}
}

func NewFormatTextTool() *Tool {
	return &Tool{
		Name:        "format_text",
		Description: "Applies various text transformation operations to the provided input text. The tool supports multiple operations including case conversion, character reversal, whitespace trimming, and text replacement. Each operation is performed independently; only one operation is applied per tool call based on the 'operation' parameter. This tool is useful for data cleaning, text normalization, string manipulation, or preparing text for display or processing. The tool handles Unicode characters correctly for all operations including reversal. For the replace operation, all occurrences of the old text are replaced with the new text throughout the entire input string. If no operation is specified, the text is returned unchanged.",
		Parameters: map[string]Parameter{
			"text": {
				Type:        "string",
				Description: "The text content to be formatted or transformed. This can be any UTF-8 text including letters, numbers, symbols, spaces, newlines, and Unicode characters. The text will be processed according to the specified operation. Examples: 'Hello World', '  Leading and trailing spaces  ', 'REVERSE ME', 'replace this word'. Empty strings are valid inputs.",
				Required:    true,
			},
			"operation": {
				Type:        "string",
				Description: "The type of transformation operation to perform on the text. Supported operations: 'uppercase' (converts all letters to uppercase), 'lowercase' (converts all letters to lowercase), 'title' (converts first letter of each word to uppercase), 'reverse' (reverses the order of all characters), 'trim' (removes leading and trailing whitespace), 'replace' (replaces occurrences of old text with new text). Operation matching is case-insensitive. If not specified, the text is returned unchanged.",
				Required:    false,
			},
			"replace_old": {
				Type:        "string",
				Description: "The text to be replaced. This parameter is required only when the operation is set to 'replace'. All occurrences of this text in the input will be replaced with the text specified in 'replace_new'. The replacement is case-sensitive and applies to the entire input string. Examples: 'word', ' ', '-', 'pattern'. If not provided when operation is 'replace', an error is returned.",
				Required:    false,
			},
			"replace_new": {
				Type:        "string",
				Description: "The replacement text that will replace all occurrences of 'replace_old'. This parameter is required only when the operation is set to 'replace'. The replacement can be any text, including an empty string (which effectively removes the old text). Examples: 'new word', '_', 'replacement'. If not provided when operation is 'replace', an error is returned.",
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
	}
}

func NewBase64EncodeTool() *Tool {
	return &Tool{
		Name:        "base64_encode",
		Description: "Encodes the provided text or data into Base64 format using the standard Base64 encoding scheme. Base64 encoding is commonly used for encoding binary data in text format, transmitting data over channels that only support text, or including binary data in text-based formats like JSON or XML. The tool uses the standard Base64 alphabet with padding ('=' characters) as defined in RFC 4648. The encoded output consists only of ASCII characters (A-Z, a-z, 0-9, +, /, and =) making it safe for transmission over protocols that may corrupt binary data. This tool is useful for preparing data for HTTP requests, encoding credentials, embedding images in text, or any scenario where binary data needs to be represented as text.",
		Parameters: map[string]Parameter{
			"data": {
				Type:        "string",
				Description: "The text or data to be encoded into Base64 format. This can be any UTF-8 text including letters, numbers, symbols, spaces, and Unicode characters. The data is treated as UTF-8 encoded bytes before Base64 encoding. Examples: 'Hello World', 'special chars: !@#$%^&*()', 'Unicode: 你好世界', 'JSON: {\"key\": \"value\"}'. The encoded output will be longer than the input (approximately 33% larger) due to Base64 encoding overhead.",
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
	}
}

func NewBase64DecodeTool() *Tool {
	return &Tool{
		Name:        "base64_decode",
		Description: "Decodes Base64 encoded data back to its original text form using the standard Base64 decoding scheme. The tool expects valid Base64 encoded input using the standard alphabet (A-Z, a-z, 0-9, +, /, =) as defined in RFC 4648. Padding characters ('=') at the end of the encoded string are properly handled. This tool is useful for retrieving original data that was previously Base64 encoded, parsing Base64 encoded credentials, extracting data from Base64 encoded API responses, or processing data that was transmitted using Base64 encoding. The decoded result is interpreted as UTF-8 text and returned as a string. The tool returns an error if the input is not valid Base64 format.",
		Parameters: map[string]Parameter{
			"data": {
				Type:        "string",
				Description: "The Base64 encoded data to be decoded. Must be a valid Base64 string using the standard Base64 alphabet (A-Z, a-z, 0-9, +, /, =). Whitespace in the input is not stripped and will cause decoding to fail. Examples: 'SGVsbG8gV29ybGQ=', 'c3BlY2lhbCBjaGFyczogIUAjJCVeJigqKQ==', '5L2g5aW95LiW55WM6K+G'. The decoded output will be the original UTF-8 text that was encoded.",
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
	}
}

func NewRegexMatchTool() *Tool {
	return &Tool{
		Name:        "regex_match",
		Description: "Tests whether the provided text matches a specified regular expression pattern using Go's regular expression syntax. The tool performs a simple match check, returning a boolean-like result indicating whether the pattern was found anywhere in the text. This tool is useful for validating input formats (email addresses, phone numbers, URLs), extracting patterns from text, filtering data based on patterns, or performing complex text matching beyond simple substring search. The regular expression syntax follows Go's RE2 engine, which is a subset of Perl-like regular expression syntax. Common regex features supported include character classes (\\d for digits, \\w for word characters), quantifiers (*, +, ?, {n,m}), anchors (^ for start, $ for end), and alternation (|). The tool returns an error if the regular expression pattern is invalid.",
		Parameters: map[string]Parameter{
			"pattern": {
				Type:        "string",
				Description: "The regular expression pattern to match against the text. Must be a valid regular expression following Go's RE2 syntax. The pattern can include special characters for matching, quantifiers, anchors, and other regex features. Examples: '\\d+' (matches one or more digits), '[a-zA-Z]+' (matches one or more letters), '^https?://.*\\..*' (matches HTTP/HTTPS URLs), '^[a-z0-9._%+-]+@[a-z0-9.-]+\\.[a-z]{2,}$' (matches email addresses), '\\b(cat|dog|bird)\\b' (matches the words cat, dog, or bird as whole words). Special regex characters like ., *, +, ?, ^, $, [, ], (, ), {, }, |, \\ must be escaped with backslash for literal matching.",
				Required:    true,
			},
			"text": {
				Type:        "string",
				Description: "The text content to be tested against the regular expression pattern. The tool checks if any part of the text matches the pattern. The match is case-sensitive unless the pattern includes case-insensitive flags. The text can be any UTF-8 string including special characters and Unicode. Examples: '12345' (matches \\d+), 'Hello World' (matches [a-zA-Z]+), 'https://example.com' (matches URL pattern), 'test@email.com' (matches email pattern).",
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
	}
}

func NewJsonParseTool() *Tool {
	return &Tool{
		Name:        "json_parse",
		Description: "Parses and validates a JSON string, optionally returning the entire JSON object in formatted (pretty-printed) form or extracting a specific field value using a dot-notation path. The tool supports complex JSON structures including nested objects and arrays. When no field path is specified, the tool returns the entire JSON object with proper indentation for readability. When a field path is specified, the tool navigates through the JSON structure to extract the value at that path. This tool is useful for validating JSON data, extracting specific values from API responses, parsing configuration files, or analyzing JSON structures. The tool returns an error if the JSON is malformed, if the specified field path does not exist, or if there is an error accessing the requested field.",
		Parameters: map[string]Parameter{
			"json_string": {
				Type:        "string",
				Description: "The JSON string to be parsed and validated. Must be valid JSON following standard JSON format rules. Supports objects ({...}), arrays ([...]), strings, numbers, booleans, and null. Examples: '{\"name\": \"John\", \"age\": 30}', '[1, 2, 3, 4, 5]', '{\"user\": {\"name\": \"Alice\", \"email\": \"alice@example.com\"}, \"active\": true}', '{\"data\": {\"items\": [{\"id\": 1, \"name\": \"Item 1\"}, {\"id\": 2, \"name\": \"Item 2\"}]}}'. JSON strings must use double quotes for keys and string values. Single quotes are not valid in JSON.",
				Required:    true,
			},
			"field": {
				Type:        "string",
				Description: "Optional dot-notation path to extract a specific field value from the parsed JSON. The path consists of field names separated by dots, with array indices specified in square brackets. If not provided, the entire JSON object is returned in formatted form. Examples: 'user.name' (extracts the name field from the user object), 'data.items[0].name' (extracts the name from the first item in the items array), 'active' (extracts a top-level field), 'data.items' (extracts the entire items array). Array indices are zero-based. If the field does not exist, an error is returned.",
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
	}
}

func NewUrlEncodeTool() *Tool {
	return &Tool{
		Name:        "url_encode",
		Description: "Performs URL encoding (also known as percent-encoding) on the provided text, converting it to a format safe for use in URLs and query parameters. URL encoding replaces unsafe ASCII characters with a '%' followed by two hexadecimal digits representing the character's byte value. This is necessary because URLs have a restricted character set, and characters outside this set must be encoded to be safely transmitted. The tool encodes characters according to the application/x-www-form-urlencoded specification, which is commonly used for encoding query parameters in HTTP requests. This tool is useful for preparing query parameters, encoding user input before embedding in URLs, constructing dynamic URLs, or ensuring data is safely transmitted as part of a URL.",
		Parameters: map[string]Parameter{
			"text": {
				Type:        "string",
				Description: "The text content to be URL encoded. Can include any characters including spaces, special characters, Unicode, and symbols. Characters that are already URL-safe (letters, numbers, hyphens, underscores, and periods) are typically left unchanged. Characters that need encoding are converted to their percent-encoded equivalent. Examples: 'hello world' encodes to 'hello+world', 'user@email.com' encodes to 'user%40email.com', 'a&b=c' encodes to 'a%26b%3Dc', '你好' encodes to '%E4%BD%A0%E5%A5%BD', '!@#$%^&*()' encodes to '%21%40%23%24%25%5E%26%2A%28%29'.",
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
	}
}

func NewUrlDecodeTool() *Tool {
	return &Tool{
		Name:        "url_decode",
		Description: "Decodes a URL-encoded string back to its original plain text form. URL encoding (percent-encoding) replaces special characters with '%' followed by hexadecimal digits; this tool reverses that process. The tool decodes according to the application/x-www-form-urlencoded specification, which is the standard for decoding query parameters from HTTP requests. This tool is useful for extracting query parameter values, decoding user input from URLs, parsing URL-encoded data, or retrieving original values that were previously URL-encoded. The tool returns an error if the input contains invalid percent-encoded sequences (e.g., incomplete sequences like '%X', invalid hex digits like '%ZZ', or malformed encoding).",
		Parameters: map[string]Parameter{
			"text": {
				Type:        "string",
				Description: "The URL-encoded text to be decoded. Should contain percent-encoded sequences where '%' is followed by two hexadecimal digits representing the byte value. The plus sign (+) is decoded to a space character. Examples: 'hello+world' decodes to 'hello world', 'user%40email.com' decodes to 'user@email.com', 'a%26b%3Dc' decodes to 'a&b=c', '%E4%BD%A0%E5%A5%BD' decodes to '你好', '%21%40%23%24%25%5E%26%2A%28%29' decodes to '!@#$%^&*()'. Text that is not percent-encoded is returned unchanged.",
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
	}
}

var BuiltinTools = []*Tool{
	NewCalculateTool(),
	NewHttpGetTool(),
	NewReadFileTool(),
	NewWriteFileTool(),
	NewDeleteFileTool(),
	NewListFilesTool(),
	NewCreateDirectoryTool(),
	NewEchoTool(),
	NewSearchFilesTool(),
	NewCurrentTimeTool(),
	NewFormatTextTool(),
	NewBase64EncodeTool(),
	NewBase64DecodeTool(),
	NewRegexMatchTool(),
	NewJsonParseTool(),
	NewUrlEncodeTool(),
	NewUrlDecodeTool(),
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
