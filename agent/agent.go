package agent

import (
	"context"

	"github.com/go-react-agent/llm"
	"github.com/go-react-agent/logger"
)

type Agent interface {
	RegisterTool(tool interface{}) error
	UnregisterTool(name string) error
	Run(ctx context.Context, query string) (*ReActResponse, error)
	RunWithCallback(ctx context.Context, query string, callback func(step *Step)) (*ReActResponse, error)
	Close() error
}

func NewAgent(llm llm.LLM, config *Config, log logger.Logger) Agent {
	return NewReActAgent(llm, config, log)
}
