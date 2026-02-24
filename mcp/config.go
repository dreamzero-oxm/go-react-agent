package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Default configuration paths.
const (
	GlobalConfigPath  = ".go-react-agent/mcp.json"
	ProjectConfigPath = ".go-react-agent/mcp.json"
)

// Config represents the MCP configuration structure.
type Config struct {
	MCPServers map[string]ServerConfig `json:"mcpServers"`
}

// ServerConfig represents the configuration for a single MCP server.
type ServerConfig struct {
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Transport string            `json:"transport,omitempty"`
	URL       string            `json:"url,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Disabled  bool              `json:"disabled,omitempty"`
	Timeout   int               `json:"timeout,omitempty"`
}

// LoadConfig loads the MCP configuration from project and global files.
// Project configuration takes precedence over global configuration.
//
// Returns:
//   - *Config: The loaded configuration.
//   - error: An error if loading fails.
func LoadConfig() (*Config, error) {
	projectConfig, projectExists := findProjectConfig()
	globalConfig, globalExists := findGlobalConfig()

	config := &Config{
		MCPServers: make(map[string]ServerConfig),
	}

	if projectExists {
		projectData, err := os.ReadFile(projectConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to read project config: %w", err)
		}
		if err := json.Unmarshal(projectData, config); err != nil {
			return nil, fmt.Errorf("failed to parse project config: %w", err)
		}
	}

	if globalExists {
		globalData, err := os.ReadFile(globalConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to read global config: %w", err)
		}

		var globalConf Config
		if err := json.Unmarshal(globalData, &globalConf); err != nil {
			return nil, fmt.Errorf("failed to parse global config: %w", err)
		}

		for name, server := range globalConf.MCPServers {
			if _, exists := config.MCPServers[name]; !exists {
				config.MCPServers[name] = server
			}
		}
	}

	return config, nil
}

// findProjectConfig locates the project-level configuration file.
//
// Returns:
//   - string: The path to the config file.
//   - bool: True if the file exists.
func findProjectConfig() (string, bool) {
	wd, err := os.Getwd()
	if err != nil {
		return "", false
	}

	configPath := filepath.Join(wd, ProjectConfigPath)
	if _, err := os.Stat(configPath); err == nil {
		return configPath, true
	}

	return "", false
}

// findGlobalConfig locates the global configuration file in the user's home directory.
//
// Returns:
//   - string: The path to the config file.
//   - bool: True if the file exists.
func findGlobalConfig() (string, bool) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}

	configPath := filepath.Join(homeDir, GlobalConfigPath)
	if _, err := os.Stat(configPath); err == nil {
		return configPath, true
	}

	return "", false
}

// Save writes the current configuration to the appropriate file.
// It prioritizes the project config if it exists, otherwise it defaults to global config.
//
// Returns:
//   - error: An error if saving fails.
func (c *Config) Save() error {
	projectConfigPath, projectExists := findProjectConfig()

	if !projectExists {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		configDir := filepath.Join(homeDir, ".go-react-agent")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		projectConfigPath = filepath.Join(homeDir, GlobalConfigPath)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(projectConfigPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// AddServer adds a new server to the configuration.
//
// Parameters:
//   - name: The server name.
//   - config: The server configuration.
//
// Returns:
//   - error: An error if the server already exists.
func (c *Config) AddServer(name string, config ServerConfig) error {
	if _, exists := c.MCPServers[name]; exists {
		return fmt.Errorf("server '%s' already exists", name)
	}

	c.MCPServers[name] = config
	return nil
}

// RemoveServer removes a server from the configuration.
//
// Parameters:
//   - name: The server name.
//
// Returns:
//   - error: An error if the server does not exist.
func (c *Config) RemoveServer(name string) error {
	if _, exists := c.MCPServers[name]; !exists {
		return fmt.Errorf("server '%s' not found", name)
	}

	delete(c.MCPServers, name)
	return nil
}

// EnableServer enables a disabled server.
//
// Parameters:
//   - name: The server name.
//
// Returns:
//   - error: An error if the server does not exist.
func (c *Config) EnableServer(name string) error {
	server, exists := c.MCPServers[name]
	if !exists {
		return fmt.Errorf("server '%s' not found", name)
	}

	server.Disabled = false
	c.MCPServers[name] = server
	return nil
}

// DisableServer disables an enabled server.
//
// Parameters:
//   - name: The server name.
//
// Returns:
//   - error: An error if the server does not exist.
func (c *Config) DisableServer(name string) error {
	server, exists := c.MCPServers[name]
	if !exists {
		return fmt.Errorf("server '%s' not found", name)
	}

	server.Disabled = true
	c.MCPServers[name] = server
	return nil
}

// GetServer retrieves the configuration for a specific server.
//
// Parameters:
//   - name: The server name.
//
// Returns:
//   - ServerConfig: The server configuration.
//   - bool: True if the server exists.
func (c *Config) GetServer(name string) (ServerConfig, bool) {
	server, exists := c.MCPServers[name]
	return server, exists
}

// ListServers returns a list of all configured server names.
//
// Returns:
//   - []string: A slice of server names.
func (c *Config) ListServers() []string {
	servers := make([]string, 0, len(c.MCPServers))
	for name := range c.MCPServers {
		servers = append(servers, name)
	}
	return servers
}

// GetEnabledServers returns a map of all enabled server configurations.
//
// Returns:
//   - map[string]ServerConfig: A map of enabled servers.
func (c *Config) GetEnabledServers() map[string]ServerConfig {
	enabled := make(map[string]ServerConfig)
	for name, server := range c.MCPServers {
		if !server.Disabled {
			enabled[name] = server
		}
	}
	return enabled
}

// ExpandEnvVars expands environment variables in the configuration.
//
// Returns:
//   - error: Always returns nil.
func (c *Config) ExpandEnvVars() error {
	for name, server := range c.MCPServers {
		if server.URL != "" {
			server.URL = os.ExpandEnv(server.URL)
		}

		for key, value := range server.Env {
			server.Env[key] = os.ExpandEnv(value)
		}

		for key, value := range server.Headers {
			server.Headers[key] = os.ExpandEnv(value)
		}

		for i, arg := range server.Args {
			server.Args[i] = os.ExpandEnv(arg)
		}

		c.MCPServers[name] = server
	}

	return nil
}

// NewDefaultConfig creates a new empty configuration.
//
// Returns:
//   - *Config: The new configuration.
func NewDefaultConfig() *Config {
	return &Config{
		MCPServers: make(map[string]ServerConfig),
	}
}

// ParseEnvMap parses a comma-separated string of KEY=VALUE pairs into a map.
//
// Parameters:
//   - envStr: The string to parse.
//
// Returns:
//   - map[string]string: The parsed map.
func ParseEnvMap(envStr string) map[string]string {
	envMap := make(map[string]string)
	if envStr == "" {
		return envMap
	}

	pairs := strings.Split(envStr, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			envMap[kv[0]] = kv[1]
		}
	}

	return envMap
}

// ParseHeaderMap parses a comma-separated string of KEY:VALUE pairs into a map.
//
// Parameters:
//   - headerStr: The string to parse.
//
// Returns:
//   - map[string]string: The parsed map.
func ParseHeaderMap(headerStr string) map[string]string {
	headerMap := make(map[string]string)
	if headerStr == "" {
		return headerMap
	}

	pairs := strings.Split(headerStr, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) == 2 {
			headerMap[kv[0]] = kv[1]
		}
	}

	return headerMap
}

// ParseArgs parses a comma-separated string into a slice of strings.
//
// Parameters:
//   - argsStr: The string to parse.
//
// Returns:
//   - []string: The parsed slice.
func ParseArgs(argsStr string) []string {
	if argsStr == "" {
		return []string{}
	}

	return strings.Split(argsStr, ",")
}
