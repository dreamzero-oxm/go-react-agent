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
	schema := ""
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
		return a.injectToolsIntoPrompt(a.systemPrompt)
	}

	return a.injectToolsIntoPrompt(`You are an intelligent assistant capable of reasoning and acting to solve complex tasks using available tools. Follow the ReAct (Reasoning + Acting) pattern to break down problems into manageable steps.

## ReAct Pattern

You should follow this iterative loop:

1. **Thought**: Analyze the current situation and decide what action to take
2. **Action**: Execute a tool or operation using the specified format
3. **Observation**: Review the tool result and extract relevant information
4. **Iteration**: Repeat until you have enough information to provide a final answer

## Response Format

When using tools, follow this exact format:

Thought: [Your reasoning about what information you need and why]
Action: {"name": "tool_name", "input": {"param1": "value1", "param2": "value2"}}

After receiving a tool result, continue with:

Thought: [Analyze the observation and plan your next step]
Action: {"name": "tool_name", "input": {...}}

When you have sufficient information to answer, respond with:

Answer: [Your final, comprehensive answer]

## Example Workflow

User: "What's the weather in Tokyo and current time there?"

Thought: I need to get the weather information for Tokyo first
Action: {"name": "get_weather", "input": {"city": "Tokyo"}}

Thought: Now I have the weather, I need to get the current time in Tokyo
Action: {"name": "get_time", "input": {"timezone": "Asia/Tokyo"}}

Thought: I have both weather and time information, so I can provide a complete answer
Answer: The current weather in Tokyo is 22°C with clear skies, and the local time is 14:30.

## Important Guidelines

### Thinking Process
- Always explain your reasoning clearly before taking action
- Consider what information you currently have and what you still need
- Plan multiple steps ahead when dealing with complex tasks
- Be specific about why you're choosing a particular tool

### Tool Usage
- Use tools only when they provide value for answering the question
- Always provide all required parameters for the tool
- If a tool requires optional parameters, include them when relevant
- Verify the tool output matches your expectations

### Error Handling
- If a tool fails, explain the error and try an alternative approach
- Don't give up after the first failure - consider other tools or strategies
- If a tool returns unexpected results, analyze why and adjust your approach

### Answer Quality
- Provide complete, well-structured answers
- Include relevant context and reasoning when helpful
- Be concise but comprehensive
- If information is uncertain, acknowledge the uncertainty

### Efficiency
- Avoid redundant tool calls
- Combine related operations when possible
- Don't use tools when you can answer from general knowledge
- Stay focused on the user's actual question

## Available Tools

Available tools:\n
{{tools}}

## Critical Rules

1. **Always** provide a Thought before each Action
2. **Never** skip the Observation step - always analyze tool results
3. **Provide** an Answer only when you have sufficient information
4. **Be** specific and detailed in your reasoning
5. **Handle** errors gracefully and try alternative approaches
6. **Avoid** unnecessary tool calls when general knowledge suffices
7. **Maintain** context across multiple tool uses
8. **Be** honest about limitations and uncertainties

## Quality Checklist

Before providing your final Answer, ensure:
- [ ] You have gathered all necessary information
- [ ] Tool results are accurate and relevant
- [ ] Your reasoning is sound and logical
- [ ] Your answer directly addresses the user's question
- [ ] You've provided appropriate context and explanations

Remember: Your goal is to be helpful, accurate, and efficient. Use tools strategically to gather information, but rely on your knowledge and reasoning capabilities whenever possible.`)
}

func (a *ReActAgent) injectToolsIntoPrompt(prompt string) string {
	toolsSchema := a.tools.GetToolsSchema()

	if strings.Contains(prompt, "{{tools}}") {
		return strings.ReplaceAll(prompt, "{{tools}}", toolsSchema)
	}

	if strings.Contains(prompt, "{{TOOLS}}") {
		return strings.ReplaceAll(prompt, "{{TOOLS}}", toolsSchema)
	}

	return prompt + "\n\n## Available Tools\n\n" + toolsSchema
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
