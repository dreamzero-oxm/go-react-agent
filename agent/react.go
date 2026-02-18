package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-react-agent/llm"
	"github.com/go-react-agent/logger"
)

type ToolRegistry interface {
	RegisterTool(tool *Tool) error
	UnregisterTool(name string) error
	Get(name string) (*Tool, error)
	List() []string
	Execute(name string, input map[string]interface{}) (string, error)
	GetToolsSchema() string
}

type ReActAgent struct {
	llm          llm.LLM
	tools        ToolRegistry
	config       *Config
	logger       logger.Logger
	systemPrompt string
}

func NewReActAgent(llm llm.LLM, config *Config, log logger.Logger) *ReActAgent {
	if config == nil {
		config = DefaultConfig()
	}

	return &ReActAgent{
		llm:    llm,
		tools:  &simpleToolRegistry{tools: make(map[string]*Tool)},
		config: config,
		logger: log,
	}
}

func NewReActAgentWithRegistry(llm llm.LLM, config *Config, log logger.Logger, registry ToolRegistry) *ReActAgent {
	if config == nil {
		config = DefaultConfig()
	}

	return &ReActAgent{
		llm:    llm,
		tools:  registry,
		config: config,
		logger: log,
	}
}

type simpleToolRegistry struct {
	tools map[string]*Tool
}

func (s *simpleToolRegistry) RegisterTool(tool *Tool) error {
	if tool.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	if tool.Execute == nil {
		return fmt.Errorf("tool execute function cannot be nil")
	}
	if _, exists := s.tools[tool.Name]; exists {
		return fmt.Errorf("tool '%s' already registered", tool.Name)
	}
	s.tools[tool.Name] = tool
	return nil
}

func (s *simpleToolRegistry) UnregisterTool(name string) error {
	if _, exists := s.tools[name]; !exists {
		return fmt.Errorf("tool '%s' not found", name)
	}
	delete(s.tools, name)
	return nil
}

func (s *simpleToolRegistry) Get(name string) (*Tool, error) {
	tool, exists := s.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}
	return tool, nil
}

func (s *simpleToolRegistry) List() []string {
	names := make([]string, 0, len(s.tools))
	for name := range s.tools {
		names = append(names, name)
	}
	return names
}

func (s *simpleToolRegistry) Execute(name string, input map[string]interface{}) (string, error) {
	tool, err := s.Get(name)
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

func (s *simpleToolRegistry) GetToolsSchema() string {
	schema := "Available tools:\n"
	for name, tool := range s.tools {
		schema += fmt.Sprintf("- %s: %s\n", name, tool.Description)
		if len(tool.Parameters) > 0 {
			schema += "  Parameters:\n"
			for paramName, param := range tool.Parameters {
				required := ""
				if param.Required {
					required = " (required)"
				}
				schema += fmt.Sprintf("    - %s (%s)%s: %s\n", paramName, param.Type, required, param.Description)
			}
		}
	}
	return schema
}

func (a *ReActAgent) SetSystemPrompt(prompt string) {
	a.systemPrompt = prompt
}

func (a *ReActAgent) GetSystemPrompt() string {
	if a.systemPrompt != "" {
		return a.systemPrompt
	}

	return `You are a helpful assistant with access to various tools. You can use tools to gather information or perform actions to help answer questions.

When you need to use a tool, follow this format:
Thought: [your reasoning about what you need to do]
Action: {"name": "tool_name", "input": {"param1": "value1", "param2": "value2"}}

After receiving the tool result, continue with:
Observation: [the tool result]
Thought: [your next reasoning]
Action: {"name": "tool_name", "input": {...}}

When you have enough information to answer, respond with:
Answer: [your final answer]

Available tools:
` + a.tools.GetToolsSchema() + `

Rules:
1. Always provide a clear Thought before each Action
2. Use tools when you need to gather information or perform actions
3. Provide an Answer when you have sufficient information
4. Be concise and direct in your reasoning
5. If a tool fails, explain why and try a different approach`
}

func (a *ReActAgent) RegisterTool(tool interface{}) error {
	if toolPtr, ok := tool.(*Tool); ok {
		return a.tools.RegisterTool(toolPtr)
	}
	return fmt.Errorf("tool must be *Tool type")
}

func (a *ReActAgent) UnregisterTool(name string) error {
	return a.tools.UnregisterTool(name)
}

func (a *ReActAgent) Register(tool *Tool) error {
	return a.tools.RegisterTool(tool)
}

func (a *ReActAgent) Get(name string) (interface{}, error) {
	return a.tools.Get(name)
}

func (a *ReActAgent) Execute(name string, input map[string]interface{}) (string, error) {
	return a.tools.Execute(name, input)
}

func (a *ReActAgent) List() []string {
	return a.tools.List()
}

func (a *ReActAgent) Run(ctx context.Context, query string) (*ReActResponse, error) {
	return a.RunWithCallback(ctx, query, nil)
}

func (a *ReActAgent) RunWithCallback(ctx context.Context, query string, callback func(step *Step)) (*ReActResponse, error) {
	a.logger.Info("Starting ReAct agent", map[string]interface{}{
		"query":          query,
		"max_iterations": a.config.MaxIterations,
	})

	messages := []llm.Message{
		{
			Role:    llm.RoleUser,
			Content: query,
		},
	}

	steps := make([]Step, 0)
	thoughts := make([]Thought, 0)

	timeoutCtx, cancel := context.WithTimeout(ctx, a.config.Timeout)
	defer cancel()

	for iteration := 0; iteration < a.config.MaxIterations; iteration++ {
		select {
		case <-timeoutCtx.Done():
			a.logger.Warn("Agent timeout reached", map[string]interface{}{
				"iteration": iteration,
			})
			return &ReActResponse{
				Thoughts: thoughts,
				Done:     false,
			}, fmt.Errorf("agent timeout after %v", a.config.Timeout)
		default:
		}

		a.logger.Debug("Iteration start", map[string]interface{}{
			"iteration": iteration + 1,
		})

		response, err := a.llm.GenerateWithSystem(a.GetSystemPrompt(), messages)
		if err != nil {
			a.logger.Error("LLM generation failed", map[string]interface{}{
				"error": err.Error(),
			})
			return nil, fmt.Errorf("LLM generation failed: %w", err)
		}

		a.logger.Debug("LLM response received", map[string]interface{}{
			"response_length": len(response),
		})

		parsed, err := a.parseResponse(response)
		if err != nil {
			a.logger.Error("Failed to parse LLM response", map[string]interface{}{
				"error": err.Error(),
			})
			return nil, fmt.Errorf("failed to parse LLM response: %w", err)
		}

		thoughts = append(thoughts, parsed.Thoughts...)

		if parsed.Done {
			a.logger.Info("Agent completed successfully", map[string]interface{}{
				"iterations": iteration + 1,
			})
			return &ReActResponse{
				Thoughts: parsed.Thoughts,
				Answer:   parsed.Answer,
				Done:     true,
			}, nil
		}

		if parsed.Action != nil {
			latestThought := parsed.Thoughts[len(parsed.Thoughts)-1]
			step := &Step{
				Thought: &latestThought,
				Action:  parsed.Action,
			}

			a.logger.Info("Executing action", map[string]interface{}{
				"tool":  parsed.Action.Name,
				"input": parsed.Action.Input,
			})

			result, err := a.tools.Execute(parsed.Action.Name, parsed.Action.Input)
			if err != nil {
				step.Error = err.Error()
				a.logger.Error("Tool execution failed", map[string]interface{}{
					"tool":  parsed.Action.Name,
					"error": err.Error(),
				})
			} else {
				step.Observation = &Observation{Content: result}
				a.logger.Debug("Tool execution succeeded", map[string]interface{}{
					"tool":   parsed.Action.Name,
					"result": result,
				})
			}

			steps = append(steps, *step)

			observationMsg := fmt.Sprintf("Observation: %s", result)
			if err != nil {
				observationMsg = fmt.Sprintf("Observation: Error - %s", err.Error())
			}

			messages = append(messages, llm.Message{
				Role:    llm.RoleUser,
				Content: observationMsg,
			})

			if callback != nil {
				callback(step)
			}
		}
	}

	a.logger.Warn("Max iterations reached", map[string]interface{}{
		"iterations": a.config.MaxIterations,
	})

	return &ReActResponse{
		Thoughts: thoughts,
		Done:     false,
	}, fmt.Errorf("max iterations (%d) reached without completion", a.config.MaxIterations)
}

func (a *ReActAgent) parseResponse(response string) (*ReActResponse, error) {
	result := &ReActResponse{
		Thoughts: make([]Thought, 0),
	}

	lines := strings.Split(response, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "Thought:") {
			thoughtContent := strings.TrimSpace(strings.TrimPrefix(line, "Thought:"))
			result.Thoughts = append(result.Thoughts, Thought{
				Content: thoughtContent,
			})
		} else if strings.HasPrefix(line, "Action:") {
			actionJSON := strings.TrimSpace(strings.TrimPrefix(line, "Action:"))
			action, err := a.parseAction(actionJSON)
			if err != nil {
				return nil, fmt.Errorf("failed to parse action: %w", err)
			}
			result.Action = action
		} else if strings.HasPrefix(line, "Answer:") {
			answer := strings.TrimSpace(strings.TrimPrefix(line, "Answer:"))
			result.Answer = answer
			result.Done = true
		}
	}

	if len(result.Thoughts) == 0 && result.Action == nil && result.Answer == "" {
		return nil, fmt.Errorf("invalid response format")
	}

	return result, nil
}

func (a *ReActAgent) parseAction(actionStr string) (*Action, error) {
	var action struct {
		Name  string                 `json:"name"`
		Input map[string]interface{} `json:"input"`
	}

	if err := json.Unmarshal([]byte(actionStr), &action); err != nil {
		return nil, fmt.Errorf("failed to unmarshal action JSON: %w", err)
	}

	if action.Name == "" {
		return nil, fmt.Errorf("action name is required")
	}

	if action.Input == nil {
		action.Input = make(map[string]interface{})
	}

	return &Action{
		Name:  action.Name,
		Input: action.Input,
	}, nil
}

func (a *ReActAgent) Stream(ctx context.Context, query string, callback func(chunk string)) error {
	a.logger.Info("Starting streaming ReAct agent", map[string]interface{}{
		"query": query,
	})

	messages := []llm.Message{
		{
			Role:    llm.RoleUser,
			Content: query,
		},
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, a.config.Timeout)
	defer cancel()

	for iteration := 0; iteration < a.config.MaxIterations; iteration++ {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("agent timeout after %v", a.config.Timeout)
		default:
		}

		err := a.llm.Stream(messages, func(chunk string) {
			callback(chunk)
		})

		if err != nil {
			return fmt.Errorf("LLM streaming failed: %w", err)
		}

		break
	}

	return nil
}

func (a *ReActAgent) Close() error {
	if a.llm != nil {
		return a.llm.Close()
	}
	if a.logger != nil {
		return a.logger.Close()
	}
	return nil
}
