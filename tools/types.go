// Package tools provides built-in tools and tool management for agents.
package tools

import (
	"fmt"
	"strings"
)

// ToolRegistry manages a collection of tools for agent use.
type ToolRegistry struct {
	tools map[string]*Tool
}

// NewToolRegistry creates a new empty tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*Tool),
	}
}

// RegisterTool registers a new tool with the registry.
func (tr *ToolRegistry) RegisterTool(tool *Tool) error {
	if tool.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	if tool.Execute == nil {
		return fmt.Errorf("tool execute function cannot be nil")
	}
	if _, exists := tr.tools[tool.Name]; exists {
		return fmt.Errorf("tool '%s' already registered", tool.Name)
	}

	tr.tools[tool.Name] = tool
	return nil
}

// Register is an alias for RegisterTool.
func (tr *ToolRegistry) Register(tool interface{}) error {
	toolStruct, ok := tool.(*Tool)
	if !ok {
		return fmt.Errorf("tool must be *Tool type")
	}
	return tr.RegisterTool(toolStruct)
}

// Unregister removes a tool from the registry by name.
func (tr *ToolRegistry) Unregister(name string) error {
	if _, exists := tr.tools[name]; !exists {
		return fmt.Errorf("tool '%s' not found", name)
	}

	delete(tr.tools, name)
	return nil
}

// UnregisterTool is an alias for Unregister.
func (tr *ToolRegistry) UnregisterTool(name string) error {
	return tr.Unregister(name)
}

// Get retrieves a tool from the registry by name.
func (tr *ToolRegistry) Get(name string) (*Tool, error) {
	tool, exists := tr.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}
	return tool, nil
}

// List returns the names of all registered tools.
func (tr *ToolRegistry) List() []string {
	names := make([]string, 0, len(tr.tools))
	for name := range tr.tools {
		names = append(names, name)
	}
	return names
}

// Execute runs a tool with the given input parameters.
//
// It validates required parameters before execution.
func (tr *ToolRegistry) Execute(name string, input map[string]interface{}) (string, error) {
	tool, err := tr.Get(name)
	if err != nil {
		return "", err
	}

	if tool.Parameters != nil {
		for paramName, param := range tool.Parameters {
			if param.Required {
				if _, exists := input[paramName]; !exists {
					return "", fmt.Errorf("missing required parameter: %s", paramName)
				}
			}
		}
	}

	return tool.Execute(input)
}

// GetToolsSchema returns a formatted string describing all available tools.
func (tr *ToolRegistry) GetToolsSchema() string {
	var builder strings.Builder
	builder.WriteString("Available tools:\n")
	for name, tool := range tr.tools {
		fmt.Fprintf(&builder, "- %s: %s\n", name, tool.Description)
		if len(tool.Parameters) > 0 {
			builder.WriteString("  Parameters:\n")
			for paramName, param := range tool.Parameters {
				required := ""
				if param.Required {
					required = " (required)"
				}
				fmt.Fprintf(&builder, "    - %s (%s)%s: %s\n", paramName, param.Type, required, param.Description)
			}
		}
	}
	return builder.String()
}
