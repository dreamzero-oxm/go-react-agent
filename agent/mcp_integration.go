package agent

import (
	"fmt"

	"github.com/dreamzero-oxm/go-react-agent/llm"
	"github.com/dreamzero-oxm/go-react-agent/logger"
	"github.com/dreamzero-oxm/go-react-agent/mcp"
)

// WithMCPIntegration adds MCP (Model Context Protocol) integration to the agent.
//
// This function enables the agent to automatically discover and use tools from
// configured MCP servers. The MCP servers are loaded from configuration files
// and their tools are automatically registered with the agent's tool registry.
//
// Returns:
//   - error: An error if enabling MCP fails.
//
// Usage:
//
//	agent := agent.NewReActAgent(llm, config, log)
//	if err := agent.WithMCPIntegration(); err != nil {
//	    log.Fatalf("Failed to enable MCP: %v", err)
//	}
func (a *ReActAgent) WithMCPIntegration() error {
	if a.config.MCPConfig == nil || !a.config.MCPConfig.Enabled {
		return fmt.Errorf("MCP integration is not enabled in config")
	}

	config, err := mcp.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load MCP config: %w", err)
	}

	manager := mcp.NewManager(config)

	if err := manager.Start(); err != nil {
		return fmt.Errorf("failed to start MCP manager: %w", err)
	}

	a.logger.Info("MCP integration started", nil)

	registry := manager.GetRegistry()
	for _, tool := range registry.GetTools() {
		if err := a.RegisterTool(tool); err != nil {
			a.logger.Warn("Failed to register MCP tool", map[string]interface{}{
				"tool":  tool.Name,
				"error": err.Error(),
			})
		} else {
			a.logger.Info("Registered MCP tool", map[string]interface{}{
				"tool": tool.Name,
			})
		}
	}

	a.logger.Info("MCP tools registered", map[string]interface{}{
		"count": len(registry.ListToolNames()),
	})

	return nil
}

// RegisterMCPTools registers tools from an MCP manager to the agent.
//
// This allows manual control over MCP integration, giving you the ability
// to start the manager yourself and register only specific tools.
//
// Parameters:
//   - manager: The MCP manager instance.
//
// Returns:
//   - error: An error if registration fails.
func (a *ReActAgent) RegisterMCPTools(manager *mcp.Manager) error {
	registry := manager.GetRegistry()

	toolCount := 0
	for _, tool := range registry.GetTools() {
		if err := a.RegisterTool(tool); err != nil {
			a.logger.Warn("Failed to register MCP tool", map[string]interface{}{
				"tool":  tool.Name,
				"error": err.Error(),
			})
		} else {
			a.logger.Info("Registered MCP tool", map[string]interface{}{
				"tool": tool.Name,
			})
			toolCount++
		}
	}

	a.logger.Info("Registered MCP tools", map[string]interface{}{
		"count": toolCount,
	})
	return nil
}

// NewAgentWithMCP creates a new agent with MCP integration automatically enabled.
//
// This is a convenience function that creates an agent and enables MCP integration
// in one step. MCP config is automatically loaded from configuration files.
//
// Parameters:
//   - llm: The LLM instance.
//   - config: The agent configuration.
//   - log: The logger instance.
//
// Returns:
//   - *ReActAgent: The created agent.
//   - error: An error if creation or MCP initialization fails.
//
// Usage:
//
//	config := agent.DefaultConfig()
//	config.MCPConfig.Enabled = true
//
//	agent, err := agent.NewAgentWithMCP(llm, config, log)
//	if err != nil {
//	    log.Fatalf("Failed to create agent: %v", err)
//	}
func NewAgentWithMCP(llm llm.LLM, config *Config, log logger.Logger) (*ReActAgent, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if config.MCPConfig == nil {
		config.MCPConfig = &MCPConfig{
			Enabled:        true,
			AutoLoadConfig: true,
		}
	} else {
		config.MCPConfig.Enabled = true
	}

	agent := NewReActAgent(llm, config, log)

	if err := agent.WithMCPIntegration(); err != nil {
		return nil, err
	}

	return agent, nil
}

// WithMCPManager creates an agent with a specific MCP manager.
//
// This gives you full control over MCP server management and tool registration.
// You can start the manager manually and register only the tools you want.
//
// Parameters:
//   - manager: The MCP manager instance.
//
// Returns:
//   - error: An error if registration fails.
//
// Usage:
//
//	manager := mcp.NewManager(config)
//	if err := manager.Start(); err != nil {
//	    log.Fatalf("Failed to start MCP manager: %v", err)
//	}
//
//	agent := agent.NewReActAgent(llm, config, log)
//	if err := agent.WithMCPManager(manager); err != nil {
//	    log.Fatalf("Failed to register MCP tools: %v", err)
//	}
func (a *ReActAgent) WithMCPManager(manager *mcp.Manager) error {
	return a.RegisterMCPTools(manager)
}

// GetMCPStatus returns the status of MCP integration.
//
// This function loads the MCP configuration and returns information about
// configured servers and their status.
//
// Returns:
//   - []mcp.ServerStatus: The status of all configured servers.
//   - error: An error if loading config or starting manager fails.
func GetMCPStatus() ([]mcp.ServerStatus, error) {
	config, err := mcp.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load MCP config: %w", err)
	}

	manager := mcp.NewManager(config)

	if err := manager.Start(); err != nil {
		return nil, fmt.Errorf("failed to start MCP manager: %w", err)
	}
	defer manager.Stop()

	return manager.GetStatus(), nil
}

// ConfigureMCP enables MCP integration with custom configuration.
//
// This function modifies the agent's configuration to enable MCP integration
// and optionally sets a custom config path.
//
// Parameters:
//   - enabled: Whether to enable MCP.
//   - configPath: Custom path to the MCP configuration file.
func (a *ReActAgent) ConfigureMCP(enabled bool, configPath string) {
	if a.config.MCPConfig == nil {
		a.config.MCPConfig = &MCPConfig{}
	}

	a.config.MCPConfig.Enabled = enabled
	a.config.MCPConfig.AutoLoadConfig = enabled
	a.config.MCPConfig.ConfigPath = configPath
}

// IsMCPEnabled returns whether MCP integration is enabled for this agent.
//
// Returns:
//   - bool: True if MCP is enabled.
func (a *ReActAgent) IsMCPEnabled() bool {
	return a.config.MCPConfig != nil && a.config.MCPConfig.Enabled
}

// MCPContext represents the MCP context available to the agent.
type MCPContext struct {
	Resources []MCPResourceInfo
	Prompts   []MCPPromptInfo
}

// MCPResourceInfo represents information about an available MCP resource.
type MCPResourceInfo struct {
	URI         string
	Name        string
	Description string
	MimeType    string
}

// MCPPromptInfo represents information about an available MCP prompt.
type MCPPromptInfo struct {
	Name        string
	Description string
	Arguments   []string
}

// GetMCPContext retrieves the MCP context (resources and prompts) for the agent.
//
// Returns:
//   - *MCPContext: The MCP context containing available resources and prompts.
//   - error: An error if MCP is not enabled or context retrieval fails.
func (a *ReActAgent) GetMCPContext() (*MCPContext, error) {
	if !a.IsMCPEnabled() {
		return nil, fmt.Errorf("MCP integration is not enabled")
	}

	config, err := mcp.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load MCP config: %w", err)
	}

	manager := mcp.NewManager(config)

	if err := manager.Start(); err != nil {
		return nil, fmt.Errorf("failed to start MCP manager: %w", err)
	}
	defer manager.Stop()

	context := &MCPContext{
		Resources: []MCPResourceInfo{},
		Prompts:   []MCPPromptInfo{},
	}

	for _, status := range manager.GetStatus() {
		if status.Status != "running" {
			continue
		}

		client, exists := manager.GetClient(status.Name)
		if !exists {
			continue
		}

		resourcesResp, listErr := client.ListResources("")
		if listErr == nil {
			for _, resource := range resourcesResp.Resources {
				context.Resources = append(context.Resources, MCPResourceInfo{
					URI:         resource.URI,
					Name:        resource.Name,
					Description: resource.Description,
					MimeType:    resource.MimeType,
				})
			}
		}

		promptsResp, listErr := client.ListPrompts("")
		if listErr == nil {
			for _, prompt := range promptsResp.Prompts {
				argNames := make([]string, 0, len(prompt.Arguments))
				for _, arg := range prompt.Arguments {
					argNames = append(argNames, arg.Name)
				}
				context.Prompts = append(context.Prompts, MCPPromptInfo{
					Name:        prompt.Name,
					Description: prompt.Description,
					Arguments:   argNames,
				})
			}
		}
	}

	return context, nil
}
