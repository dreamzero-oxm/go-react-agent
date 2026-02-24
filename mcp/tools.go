package mcp

import (
	"fmt"

	"github.com/dreamzero-oxm/go-react-agent/mcp/protocol"
	"github.com/dreamzero-oxm/go-react-agent/tools"
)

// MCPTool adapts an MCP protocol tool to the agent's Tool interface.
type MCPTool struct {
	name    string
	client  *Client
	mcpTool *protocol.Tool
}

// NewMCPTool creates a new agent Tool from an MCP tool definition.
//
// Parameters:
//   - client: The MCP client that owns the tool.
//   - mcpTool: The MCP tool definition.
//
// Returns:
//   - *tools.Tool: The adapted agent tool.
func NewMCPTool(client *Client, mcpTool *protocol.Tool) *tools.Tool {
	mcpToolWrap := &MCPTool{
		name:    mcpTool.Name,
		client:  client,
		mcpTool: mcpTool,
	}

	return &tools.Tool{
		Name:        mcpTool.Name,
		Description: mcpTool.Description,
		Parameters:  convertInputSchema(mcpTool.InputSchema),
		Execute:     mcpToolWrap.Execute,
	}
}

// Execute calls the remote MCP tool.
//
// Parameters:
//   - input: The input arguments for the tool.
//
// Returns:
//   - string: The tool execution result (text content).
//   - error: An error if the tool call fails.
func (mt *MCPTool) Execute(input map[string]interface{}) (string, error) {
	req := &protocol.ToolCallRequest{
		Name:      mt.name,
		Arguments: input,
	}

	resp, err := mt.client.CallTool(req)
	if err != nil {
		return "", fmt.Errorf("MCP tool call failed: %w", err)
	}

	if resp.IsError {
		if len(resp.Content) > 0 {
			return "", fmt.Errorf("MCP tool returned error: %s", resp.Content[0].Text)
		}
		return "", fmt.Errorf("MCP tool returned error")
	}

	var result string
	for _, content := range resp.Content {
		if content.Type == "text" {
			result += content.Text + "\n"
		}
	}

	return result, nil
}

// convertInputSchema converts JSON Schema to the agent's parameter definition format.
//
// Parameters:
//   - schema: The JSON Schema map.
//
// Returns:
//   - map[string]tools.Parameter: The converted parameters.
func convertInputSchema(schema map[string]interface{}) map[string]tools.Parameter {
	params := make(map[string]tools.Parameter)

	if properties, ok := schema["properties"].(map[string]interface{}); ok {
		for propName, propValue := range properties {
			if propMap, ok := propValue.(map[string]interface{}); ok {
				param := tools.Parameter{
					Type:        "string",
					Description: "",
					Required:    false,
				}

				if propType, ok := propMap["type"].(string); ok {
					param.Type = propType
				}

				if propDesc, ok := propMap["description"].(string); ok {
					param.Description = propDesc
				}

				params[propName] = param
			}
		}
	}

	if requiredList, ok := schema["required"].([]interface{}); ok {
		for _, reqItem := range requiredList {
			if reqName, ok := reqItem.(string); ok {
				if param, exists := params[reqName]; exists {
					param.Required = true
					params[reqName] = param
				}
			}
		}
	}

	return params
}

// MCPToolRegistry manages MCP clients and their tools.
type MCPToolRegistry struct {
	clients map[string]*Client
	tools   map[string]*tools.Tool
}

// NewMCPToolRegistry creates a new registry.
//
// Returns:
//   - *MCPToolRegistry: The created registry.
func NewMCPToolRegistry() *MCPToolRegistry {
	return &MCPToolRegistry{
		clients: make(map[string]*Client),
		tools:   make(map[string]*tools.Tool),
	}
}

// AddClient adds an MCP client and registers its tools.
//
// Parameters:
//   - client: The initialized MCP client.
//
// Returns:
//   - error: An error if the client is not ready or tool listing fails.
func (mtr *MCPToolRegistry) AddClient(client *Client) error {
	if !client.IsReady() {
		return fmt.Errorf("client is not ready")
	}

	mtr.clients[client.Name()] = client

	toolsResp, err := client.ListTools("")
	if err != nil {
		return fmt.Errorf("failed to list tools for client %s: %w", client.Name(), err)
	}

	for _, mcpTool := range toolsResp.Tools {
		toolName := fmt.Sprintf("%s:%s", client.Name(), mcpTool.Name)
		tool := NewMCPTool(client, &mcpTool)
		tool.Name = toolName
		mtr.tools[toolName] = tool
	}

	return nil
}

// RemoveClient removes an MCP client and its tools.
//
// Parameters:
//   - name: The client name.
func (mtr *MCPToolRegistry) RemoveClient(name string) {
	delete(mtr.clients, name)

	for toolName := range mtr.tools {
		if len(toolName) > len(name) && toolName[:len(name)+1] == name+":" {
			delete(mtr.tools, toolName)
		}
	}
}

// GetTools returns all registered tools.
//
// Returns:
//   - []*tools.Tool: A slice of all registered tools.
func (mtr *MCPToolRegistry) GetTools() []*tools.Tool {
	result := make([]*tools.Tool, 0, len(mtr.tools))
	for _, tool := range mtr.tools {
		result = append(result, tool)
	}
	return result
}

// GetTool retrieves a specific tool by name.
//
// Parameters:
//   - name: The tool name.
//
// Returns:
//   - *tools.Tool: The tool.
//   - bool: True if found.
func (mtr *MCPToolRegistry) GetTool(name string) (*tools.Tool, bool) {
	tool, exists := mtr.tools[name]
	return tool, exists
}

// ListToolNames lists the names of all registered tools.
//
// Returns:
//   - []string: A list of tool names.
func (mtr *MCPToolRegistry) ListToolNames() []string {
	names := make([]string, 0, len(mtr.tools))
	for name := range mtr.tools {
		names = append(names, name)
	}
	return names
}

// ListClients lists the names of all registered clients.
//
// Returns:
//   - []string: A list of client names.
func (mtr *MCPToolRegistry) ListClients() []string {
	names := make([]string, 0, len(mtr.clients))
	for name := range mtr.clients {
		names = append(names, name)
	}
	return names
}

// GetClient retrieves a client by name.
//
// Parameters:
//   - name: The client name.
//
// Returns:
//   - *Client: The client.
//   - bool: True if found.
func (mtr *MCPToolRegistry) GetClient(name string) (*Client, bool) {
	client, exists := mtr.clients[name]
	return client, exists
}

// CloseAll closes all registered clients.
//
// Returns:
//   - error: The last error encountered during closing, if any.
func (mtr *MCPToolRegistry) CloseAll() error {
	var lastErr error
	for _, client := range mtr.clients {
		if err := client.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
