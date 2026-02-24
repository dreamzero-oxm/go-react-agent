package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dreamzero-oxm/go-react-agent/mcp/protocol"
	"github.com/dreamzero-oxm/go-react-agent/mcp/transport"
)

// Manager handles the lifecycle and management of multiple MCP servers.
type Manager struct {
	config   *Config
	registry *MCPToolRegistry
	clients  map[string]*Client
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	debug    bool
}

// ServerStatus represents the runtime status of an MCP server.
type ServerStatus struct {
	Name      string
	Type      string
	Status    string
	Error     string
	Tools     []ToolInfo
	Resources []ResourceInfo
	Prompts   []PromptInfo
	Command   string
	URL       string
}

// ToolInfo provides basic information about an available tool.
type ToolInfo struct {
	Name        string
	Description string
}

// ResourceInfo provides basic information about an available resource.
type ResourceInfo struct {
	URI         string
	Name        string
	Description string
}

// PromptInfo provides basic information about an available prompt.
type PromptInfo struct {
	Name        string
	Description string
}

// NewManager creates a new MCP Manager.
//
// Parameters:
//   - config: The initial configuration.
//
// Returns:
//   - *Manager: The created manager.
func NewManager(config *Config) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		config:   config,
		registry: NewMCPToolRegistry(),
		clients:  make(map[string]*Client),
		ctx:      ctx,
		cancel:   cancel,
		debug:    false,
	}
}

// SetDebug enables or disables debug logging.
//
// Parameters:
//   - enabled: Whether to enable debug logging.
func (m *Manager) SetDebug(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.debug = enabled
}

// isDebug checks if debug mode is enabled.
//
// Returns:
//   - bool: True if debug mode is enabled.
func (m *Manager) isDebug() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.debug
}

// debugLog prints debug messages if debug mode is enabled.
//
// Parameters:
//   - format: The format string.
//   - args: The arguments to format.
func (m *Manager) debugLog(format string, args ...interface{}) {
	m.mu.RLock()
	enabled := m.debug
	m.mu.RUnlock()

	if enabled {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

// Start initializes and starts all enabled MCP servers.
//
// Returns:
//   - error: An error if environment expansion fails.
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.config.ExpandEnvVars(); err != nil {
		return fmt.Errorf("failed to expand env vars: %w", err)
	}

	enabledServers := m.config.GetEnabledServers()

	m.mu.Unlock()
	for name, serverConfig := range enabledServers {
		if err := m.startServer(name, serverConfig); err != nil {
			fmt.Printf("Warning: failed to start server '%s': %v\n", name, err)
		}
	}
	m.mu.Lock()

	return nil
}

// startServer starts a single MCP server based on its configuration.
//
// Parameters:
//   - name: The server name.
//   - config: The server configuration.
//
// Returns:
//   - error: An error if starting the server or client fails.
func (m *Manager) startServer(name string, config ServerConfig) error {
	m.debugLog("Starting server: %s", name)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	var trans transport.Transport

	if config.Transport == "sse" || config.URL != "" {
		m.debugLog("Server '%s' using SSE transport: %s", name, config.URL)
		sseConfig := &transport.SSEConfig{
			URL:       config.URL,
			Headers:   config.Headers,
			Timeout:   config.Timeout,
			KeepAlive: true,
			Debug:     m.isDebug(),
		}
		trans = transport.NewSSETransport(sseConfig)
	} else {
		m.debugLog("Server '%s' using Stdio transport: %s %v", name, config.Command, config.Args)
		stdioConfig := &transport.StdioConfig{
			Command: config.Command,
			Args:    config.Args,
			Env:     mapSlice(config.Env),
			Timeout: config.Timeout,
			Debug:   m.isDebug(),
		}
		trans = transport.NewStdioTransport(stdioConfig)
	}

	clientConfig := &ClientConfig{
		Name:      name,
		Transport: trans,
		Debug:     m.isDebug(),
	}

	client := NewClient(clientConfig)

	m.debugLog("Server '%s': starting client...", name)
	if err := client.Start(); err != nil {
		return fmt.Errorf("failed to start client: %w", err)
	}

	m.debugLog("Server '%s': preparing initialization request", name)
	initReq := &protocol.InitializeRequest{
		ProtocolVersion: "2024-11-05",
		Capabilities: protocol.ClientCapabilities{
			Roots: &protocol.RootsCapability{
				ListChanged: false,
			},
		},
		ClientInfo: protocol.Implementation{
			Name:    "go-react-agent",
			Version: "1.0.0",
		},
	}

	type initResult struct {
		resp *protocol.InitializeResponse
		err  error
	}

	resultChan := make(chan initResult, 1)
	go func() {
		resp, err := client.Initialize(initReq)
		resultChan <- initResult{resp: resp, err: err}
	}()

	m.debugLog("Server '%s': waiting for initialization (timeout: 45s)", name)

	select {
	case result := <-resultChan:
		if result.err != nil {
			m.debugLog("Server '%s': initialization failed: %v", name, result.err)
			client.Close()
			return fmt.Errorf("failed to initialize client: %w", result.err)
		}
		m.debugLog("Server '%s': initialization succeeded", name)
	case <-ctx.Done():
		m.debugLog("Server '%s': initialization timeout", name)
		client.Close()
		return fmt.Errorf("initialization timeout for server '%s'", name)
	}

	m.clients[name] = client

	if err := m.registry.AddClient(client); err != nil {
		m.debugLog("Server '%s': failed to add client to registry: %v", name, err)
		client.Close()
		delete(m.clients, name)
		return fmt.Errorf("failed to add client to registry: %w", err)
	}

	m.debugLog("Server '%s': successfully started and added to registry", name)

	return nil
}

// mapSlice converts a map of strings to a slice of "KEY=VALUE" strings.
//
// Parameters:
//   - m: The map to convert.
//
// Returns:
//   - []string: The slice of strings.
func mapSlice(m map[string]string) []string {
	result := make([]string, 0, len(m))
	for k, v := range m {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}

// Stop stops the manager and all running servers.
//
// Returns:
//   - error: An error if closing the registry fails.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cancel()

	if err := m.registry.CloseAll(); err != nil {
		return err
	}

	for _, client := range m.clients {
		client.Close()
	}

	m.clients = make(map[string]*Client)

	return nil
}

// GetStatus returns the current status of all configured servers.
//
// Returns:
//   - []ServerStatus: A slice of server statuses.
func (m *Manager) GetStatus() []ServerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make([]ServerStatus, 0, len(m.config.MCPServers))

	for name, serverConfig := range m.config.MCPServers {
		status := ServerStatus{
			Name:  name,
			Error: "",
		}

		if serverConfig.Disabled {
			status.Status = "disabled"
			statuses = append(statuses, status)
			continue
		}

		client, exists := m.clients[name]
		if !exists {
			status.Status = "failed"
			status.Error = "client not started"
			statuses = append(statuses, status)
			continue
		}

		if !client.IsReady() {
			status.Status = "initializing"
			statuses = append(statuses, status)
			continue
		}

		status.Status = "running"

		if serverConfig.Transport == "sse" {
			status.Type = "sse"
			status.URL = serverConfig.URL
		} else {
			status.Type = "stdio"
			status.Command = serverConfig.Command
		}

		timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		type listResult struct {
			tools     *protocol.ToolListResponse
			resources *protocol.ResourceListResponse
			prompts   *protocol.PromptListResponse
		}

		resultChan := make(chan listResult, 1)
		errChan := make(chan error, 1)

		go func() {
			var result listResult

			if toolsResp, err := client.ListTools(""); err == nil {
				result.tools = toolsResp
			}

			if resourcesResp, err := client.ListResources(""); err == nil {
				result.resources = resourcesResp
			}

			if promptsResp, err := client.ListPrompts(""); err == nil {
				result.prompts = promptsResp
			}

			select {
			case resultChan <- result:
			case <-timeoutCtx.Done():
			}
		}()

		select {
		case result := <-resultChan:
			if result.tools != nil {
				for _, tool := range result.tools.Tools {
					status.Tools = append(status.Tools, ToolInfo{
						Name:        tool.Name,
						Description: tool.Description,
					})
				}
			}

			if result.resources != nil {
				for _, resource := range result.resources.Resources {
					status.Resources = append(status.Resources, ResourceInfo{
						URI:         resource.URI,
						Name:        resource.Name,
						Description: resource.Description,
					})
				}
			}

			if result.prompts != nil {
				for _, prompt := range result.prompts.Prompts {
					status.Prompts = append(status.Prompts, PromptInfo{
						Name:        prompt.Name,
						Description: prompt.Description,
					})
				}
			}
		case <-timeoutCtx.Done():
			status.Error = "timeout fetching server details"
		case err := <-errChan:
			status.Error = err.Error()
		}

		statuses = append(statuses, status)
	}

	return statuses
}

// GetRegistry returns the tool registry associated with the manager.
//
// Returns:
//   - *MCPToolRegistry: The tool registry.
func (m *Manager) GetRegistry() *MCPToolRegistry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.registry
}

// GetClient retrieves a running client by name.
//
// Parameters:
//   - name: The server name.
//
// Returns:
//   - *Client: The client instance.
//   - bool: True if found.
func (m *Manager) GetClient(name string) (*Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, exists := m.clients[name]
	return client, exists
}

// AddServer adds and starts a new server.
//
// Parameters:
//   - name: The server name.
//   - config: The server configuration.
//
// Returns:
//   - error: An error if the server is already running or fails to start.
func (m *Manager) AddServer(name string, config ServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.clients[name]; exists {
		return fmt.Errorf("server '%s' is already running", name)
	}

	if err := m.startServer(name, config); err != nil {
		return err
	}

	return nil
}

// RemoveServer stops and removes a server.
//
// Parameters:
//   - name: The server name.
//
// Returns:
//   - error: Always returns nil.
func (m *Manager) RemoveServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[name]
	if exists {
		client.Close()
		delete(m.clients, name)
	}

	m.registry.RemoveClient(name)

	return nil
}

// EnableServer enables and starts a previously disabled server.
//
// Parameters:
//   - name: The server name.
//
// Returns:
//   - error: An error if the server is not found, already enabled, or fails to start.
func (m *Manager) EnableServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	serverConfig, exists := m.config.GetServer(name)
	if !exists {
		return fmt.Errorf("server '%s' not found", name)
	}

	if serverConfig.Disabled {
		return fmt.Errorf("server '%s' is already enabled", name)
	}

	if err := m.config.EnableServer(name); err != nil {
		return err
	}

	return m.startServer(name, serverConfig)
}

// DisableServer stops and disables a running server.
//
// Parameters:
//   - name: The server name.
//
// Returns:
//   - error: An error if the server is not found.
func (m *Manager) DisableServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[name]
	if exists {
		client.Close()
		delete(m.clients, name)
	}

	m.registry.RemoveClient(name)

	return m.config.DisableServer(name)
}

// GetConfig returns the current configuration.
//
// Returns:
//   - *Config: The configuration.
func (m *Manager) GetConfig() *Config {
	return m.config
}

// SaveConfig saves the current configuration to disk.
//
// Returns:
//   - error: An error if saving fails.
func (m *Manager) SaveConfig() error {
	return m.config.Save()
}
