package agent

import (
	"context"

	"github.com/dreamzero-oxm/go-react-agent/llm"
	"github.com/dreamzero-oxm/go-react-agent/logger"
)

// Agent defines the interface for autonomous agents that can use tools.
type Agent interface {
	// RegisterTool registers a new tool with the agent.
	RegisterTool(tool interface{}) error
	// UnregisterTool removes a tool from the agent by name.
	UnregisterTool(name string) error
	// Run executes the agent with the given query and returns the response.
	Run(ctx context.Context, query string) (*ReActResponse, error)
	// RunWithCallback executes the agent with a callback function for each step.
	RunWithCallback(ctx context.Context, query string, callback func(step *Step)) (*ReActResponse, error)
	// Close closes the agent and releases any resources.
	Close() error
}

// NewAgent creates a new Agent instance with the provided LLM, configuration, and logger.
//
// It returns a ReActAgent implementation by default.
func NewAgent(llm llm.LLM, config *Config, log logger.Logger) Agent {
	return NewReActAgent(llm, config, log)
}
