package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/dreamzero-oxm/go-react-agent/llm"
	"github.com/dreamzero-oxm/go-react-agent/logger"
	"github.com/dreamzero-oxm/go-react-agent/skills"
	"github.com/dreamzero-oxm/go-react-agent/tools"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ToolRegistry defines the interface for managing and executing tools.
//
// Implementations can provide custom tool storage and execution strategies.
type ToolRegistry interface {
	// RegisterTool registers a new tool with the registry.
	RegisterTool(tool *tools.Tool) error
	// UnregisterTool removes a tool from the registry by name.
	UnregisterTool(name string) error
	// Get retrieves a tool by name.
	Get(name string) (*tools.Tool, error)
	// List returns the names of all registered tools.
	List() []string
	// Execute runs a tool with the given input parameters.
	Execute(name string, input map[string]interface{}) (string, error)
	// GetToolsSchema returns a formatted string describing all available tools.
	GetToolsSchema() string
}

// ReActAgent implements the ReAct (Reasoning + Acting) pattern for autonomous agents.
//
// It maintains a registry of tools and iteratively reasons, acts, and observes
// until it can provide a final answer to the user's query.
type ReActAgent struct {
	llm          llm.LLM                  // LLM for generating responses
	tools        ToolRegistry             // Registry of available tools
	config       *Config                  // Agent configuration
	logger       logger.Logger            // Logger for agent operations
	systemPrompt string                   // Custom system prompt (optional)
	parser       ResponseParser           // Parser for LLM responses
	skills       map[string]*skills.Skill // Loaded Claude Code Skills (name -> skill)
}

// NewReActAgent creates a new ReActAgent instance with the provided LLM, configuration, and logger.
//
// If config is nil, DefaultConfig() will be used.
// A default JSONParser is automatically configured if config.Parser is nil.
func NewReActAgent(llm llm.LLM, config *Config, log logger.Logger) *ReActAgent {
	if config == nil {
		config = DefaultConfig()
	}

	// Ensure parser is set
	if config.Parser == nil {
		config.Parser = NewJSONParser()
	}

	return &ReActAgent{
		llm:    llm,
		tools:  &simpleToolRegistry{tools: make(map[string]*tools.Tool)},
		config: config,
		logger: log,
		parser: config.Parser,
	}
}

// NewReActAgentWithRegistry creates a new ReActAgent with a custom tool registry.
//
// This allows sharing tools across multiple agents or using custom registry implementations.
func NewReActAgentWithRegistry(llm llm.LLM, config *Config, log logger.Logger, registry ToolRegistry) *ReActAgent {
	if config == nil {
		config = DefaultConfig()
	}

	if config.Parser == nil {
		config.Parser = NewJSONParser()
	}

	return &ReActAgent{
		llm:    llm,
		tools:  registry,
		config: config,
		logger: log,
		parser: config.Parser,
	}
}

// simpleToolRegistry is a basic in-memory implementation of ToolRegistry.
type simpleToolRegistry struct {
	tools map[string]*tools.Tool
}

// RegisterTool adds a tool to the registry.
func (s *simpleToolRegistry) RegisterTool(tool *tools.Tool) error {
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

// UnregisterTool removes a tool from the registry.
func (s *simpleToolRegistry) UnregisterTool(name string) error {
	if _, exists := s.tools[name]; !exists {
		return fmt.Errorf("tool '%s' not found", name)
	}
	delete(s.tools, name)
	return nil
}

// Get retrieves a tool from the registry by name.
func (s *simpleToolRegistry) Get(name string) (*tools.Tool, error) {
	tool, exists := s.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}
	return tool, nil
}

// List returns the names of all registered tools.
func (s *simpleToolRegistry) List() []string {
	names := make([]string, 0, len(s.tools))
	for name := range s.tools {
		names = append(names, name)
	}
	return names
}

// Execute runs a tool with the given input parameters.
//
// It validates required parameters before execution.
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

// GetToolsSchema returns a formatted string describing all available tools.
func (s *simpleToolRegistry) GetToolsSchema() string {
	var builder strings.Builder
	for name, tool := range s.tools {
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

// SetSystemPrompt sets a custom system prompt for the agent.
func (a *ReActAgent) SetSystemPrompt(prompt string) {
	a.systemPrompt = prompt
}

// GetSystemPrompt returns the current system prompt with tools and output format injected.
func (a *ReActAgent) GetSystemPrompt() string {
	basePrompt := a.getBaseSystemPrompt()

	// Inject structured output instructions if configured
	if a.config.Output != nil && a.config.Output.EnableStructuredOutput && a.config.Output.OutputSchema != "" {
		basePrompt = a.injectOutputFormat(basePrompt)
	}

	basePrompt = a.injectToolsIntoPrompt(basePrompt)
	basePrompt = a.injectMCPContext(basePrompt)
	basePrompt = a.injectSkillsContext(basePrompt)

	return basePrompt
}

// getBaseSystemPrompt returns the base system prompt, using custom or default.
func (a *ReActAgent) getBaseSystemPrompt() string {
	if a.systemPrompt != "" {
		return a.systemPrompt
	}

	return `You are an intelligent assistant capable of reasoning and acting to solve complex tasks using available tools. Follow the ReAct (Reasoning + Acting) pattern to break down problems into manageable steps.

## ReAct Pattern

You should follow this iterative loop:

1. **Thought**: Analyze the current situation and decide what action to take
2. **Action**: Execute a tool or operation
3. **Observation**: Review the tool result and extract relevant information
4. **Iteration**: Repeat until you have enough information to provide a final answer

## Response Format

IMPORTANT: You must respond with a valid JSON object only. Do not include any additional text or markdown formatting (except if the LLM automatically wraps in markdown code blocks).

When using tools, respond with this JSON structure:

{
  "thoughts": [
    {
      "content": "Your reasoning about what information you need and why"
    }
  ],
  "action": {
    "name": "tool_name",
    "input": {
      "param1": "value1",
      "param2": "value2"
    }
  },
  "answer": null,
  "done": false
}

After receiving a tool result, continue with:

{
  "thoughts": [
    {
      "content": "Analyze the observation and plan your next step"
    }
  ],
  "action": {
    "name": "tool_name",
    "input": {...}
  },
  "answer": null,
  "done": false
}

When you have sufficient information to answer, respond with:

{
  "thoughts": [
    {
      "content": "Your reasoning about having enough information"
    }
  ],
  "action": null,
  "answer": "Your final, comprehensive answer",
  "done": true
}

## Example Workflow

User: "What's the weather in Tokyo and current time there?"

Response:
{
  "thoughts": [{"content": "I need to get the weather information for Tokyo first"}],
  "action": {"name": "get_weather", "input": {"city": "Tokyo"}},
  "answer": null,
  "done": false
}

Response:
{
  "thoughts": [{"content": "Now I have the weather, I need to get the current time in Tokyo"}],
  "action": {"name": "get_time", "input": {"timezone": "Asia/Tokyo"}},
  "answer": null,
  "done": false
}

Response:
{
  "thoughts": [{"content": "I have both weather and time information, so I can provide a complete answer"}],
  "action": null,
  "answer": "The current weather in Tokyo is 22°C with clear skies, and the local time is 14:30.",
  "done": true
}

## Important Guidelines

### Thinking Process
- Always explain your reasoning clearly in the thoughts array
- Consider what information you currently have and what you still need
- Plan multiple steps ahead when dealing with complex tasks

### Tool Usage
- Use tools only when they provide value for answering the question
- Always provide all required parameters for the tool
- Verify the tool output matches your expectations

### Error Handling
- If a tool fails, explain the error in thoughts and try an alternative approach
- Don't give up after the first failure - consider other tools or strategies

### JSON Format Requirements
- Response must be valid JSON
- "thoughts" must be a non-empty array
- "action" and "answer" are mutually exclusive (one must be null)
- "done" must be true only when providing an answer
- You may wrap JSON in markdown code blocks if needed

## Available Tools

Available tools:\n
{{tools}}

## Critical Rules

1. **Always** provide thoughts explaining your reasoning
2. **Response must be valid JSON**
3. **NEVER** mix action and answer in the same response
4. **Be** specific and detailed in your reasoning
5. **Handle** errors gracefully and try alternative approaches
6. **Avoid** unnecessary tool calls when general knowledge suffices

Remember: Your goal is to be helpful, accurate, and efficient. Always respond with valid JSON.`
}

// injectToolsIntoPrompt injects the tools schema into the prompt.
func (a *ReActAgent) injectToolsIntoPrompt(prompt string) string {
	toolsSchema := a.tools.GetToolsSchema()

	if strings.Contains(prompt, "{{tools}}") {
		return strings.ReplaceAll(prompt, "{{tools}}", toolsSchema)
	}

	if strings.Contains(prompt, "{{TOOLS}}") {
		return strings.ReplaceAll(prompt, "{{TOOLS}}", toolsSchema)
	}

	return fmt.Sprintf("%s\n\n## Available Tools\n\n%s", prompt, toolsSchema)
}

// injectMCPContext injects MCP resources and prompts information into the prompt.
func (a *ReActAgent) injectMCPContext(prompt string) string {
	if !a.IsMCPEnabled() {
		return prompt
	}

	mcpContext, err := a.GetMCPContext()
	if err != nil {
		a.logger.Warn("Failed to get MCP context", map[string]interface{}{
			"error": err.Error(),
		})
		return prompt
	}

	var builder strings.Builder

	if len(mcpContext.Resources) > 0 {
		builder.WriteString("\n\n## Available MCP Resources\n\n")
		builder.WriteString("You can access the following resources for additional context:\n\n")
		for _, resource := range mcpContext.Resources {
			fmt.Fprintf(&builder, "- %s (%s): %s\n", resource.Name, resource.URI, resource.Description)
			if resource.MimeType != "" {
				fmt.Fprintf(&builder, "  MIME Type: %s\n", resource.MimeType)
			}
		}
	}

	if len(mcpContext.Prompts) > 0 {
		builder.WriteString("\n\n## Available MCP Prompts\n\n")
		builder.WriteString("You can use the following prompt templates for structured interactions:\n\n")
		for _, promptInfo := range mcpContext.Prompts {
			fmt.Fprintf(&builder, "- %s: %s\n", promptInfo.Name, promptInfo.Description)
			if len(promptInfo.Arguments) > 0 {
				builder.WriteString("  Arguments: ")
				for i, arg := range promptInfo.Arguments {
					if i > 0 {
						builder.WriteString(", ")
					}
					builder.WriteString(arg)
				}
				builder.WriteString("\n")
			}
		}
	}

	if builder.Len() > 0 {
		return prompt + builder.String()
	}

	return prompt
}

// injectSkillsContext injects Claude Code Skills information into the prompt.
// Only metadata is included; full skill content is loaded on-demand when agent requests it.
func (a *ReActAgent) injectSkillsContext(prompt string) string {
	if !a.IsSkillEnabled() || len(a.skills) == 0 {
		return prompt
	}

	var builder strings.Builder
	builder.WriteString("\n\n## Available Skills\n\n")
	builder.WriteString("When you need domain knowledge or guidance, you can request to use a specific skill.\n\n")
	builder.WriteString("Available skills:\n\n")

	for _, skill := range a.skills {
		builder.WriteString(fmt.Sprintf("- **%s**", skill.Name))
		if skill.Version != "" {
			builder.WriteString(fmt.Sprintf(" (v%s)", skill.Version))
		}
		builder.WriteString("\n")
		if skill.Description != "" {
			builder.WriteString(fmt.Sprintf("  - Description: %s\n", skill.Description))
		}
		if len(skill.Tags) > 0 {
			builder.WriteString(fmt.Sprintf("  - Tags: %s\n", strings.Join(skill.Tags, ", ")))
		}
	}

	builder.WriteString("\nTo use a skill, respond with an action:\n")
	builder.WriteString(`{"action": {"name": "use_skill", "input": {"skill_name": "skill-name"}}}`)

	return prompt + builder.String()
}

// handleSkillUsage handles when the agent wants to use a skill.
// It loads the full skill content and generates a response with that context.
//
// Parameters:
//   - ctx: Context for the operation.
//   - skillName: Name of the skill to use.
//   - query: The original user query.
//
// Returns:
//   - string: The LLM response with skill context.
//   - error: An error if the skill is not found or LLM call fails.
func (a *ReActAgent) handleSkillUsage(ctx context.Context, skillName string, query string) (string, error) {
	targetSkill, ok := a.skills[skillName]
	if !ok {
		return "", fmt.Errorf("skill '%s' not found", skillName)
	}

	// Log skill usage
	a.logger.Info("Agent using skill", map[string]interface{}{
		"skill": skillName,
	})

	// Create skill-enhanced prompt
	skillPrompt := fmt.Sprintf(`You are an expert assistant specialized in the following domain. Use the provided skill knowledge to answer the user's question accurately and comprehensively.

---

## Skill: %s

%s

---

## User Query
%s

## Instructions for Answering

1. **Understand the Skill Content**: Carefully review the skill documentation above. It contains specialized knowledge, best practices, and guidelines for this domain.

2. **Provide a Comprehensive Answer**:
   - Address all aspects of the user's query
   - Use specific terminology and concepts from the skill content
   - Include relevant examples when helpful
   - Explain complex concepts clearly

3. **Quality Standards**:
   - Be accurate and precise - rely on the skill content as your primary source
   - Be thorough - cover important details and considerations
   - Be practical - provide actionable advice when applicable
   - Be clear - use well-structured, easy-to-follow explanations

4. **Handling Uncertainties**:
   - If the skill content doesn't fully address the query, acknowledge this limitation
   - Provide the best possible answer based on available information
   - Suggest what additional information might be needed

5. **Output Format**:
   - Use markdown formatting for better readability
   - Organize information with clear headings and bullet points
   - Include code examples or configurations when relevant

Now, please provide a comprehensive, expert-level answer to the user's query based on the skill knowledge above.
`, targetSkill.Name, targetSkill.Content, query)

	// Get LLM response with skill context
	messages := []llm.Message{
		{Role: llm.RoleUser, Content: skillPrompt},
	}

	response, err := a.llm.Generate(messages)
	if err != nil {
		return "", fmt.Errorf("failed to get LLM response with skill: %w", err)
	}

	return response, nil
}

// injectOutputFormat injects structured output instructions into the prompt.
func (a *ReActAgent) injectOutputFormat(prompt string) string {
	if a.config.Output == nil || a.config.Output.OutputSchema == "" {
		return prompt
	}

	instructions := fmt.Sprintf(`

## Structured Output Format

CRITICAL: Your final response MUST be valid JSON ONLY - no additional text, explanations, or markdown formatting.

### Response Structure
Your response must follow this exact structure:
{
  "thoughts": [{"content": "string"}],
  "action": {"name": "tool_name", "input": {...}} | null,
  "answer": "JSON_STRING",
  "done": boolean
}

### Answer Field Schema
The content of the "answer" field (when parsed as JSON) must match this schema:

%s

### Response Requirements

1. **JSON ONLY**: Respond with valid JSON only - no introductory text, no explanations, no markdown code blocks
2. **Fixed Response Structure**: Your response must always contain the four top-level fields: thoughts, action, answer, done
3. **Answer Field Compliance**: When "done" is true, the "answer" field must contain a JSON string that, when parsed, matches the schema above
4. **Field Requirements**:
   - "thoughts": Array of thought objects with "content" field (required)
   - "action": Tool action object or null (mutually exclusive with "answer" - only one can be set)
   - "answer": JSON string containing your structured output (required when "done" is true, empty string otherwise)
   - "done": Boolean flag (required - set to true only when task is complete)

5. **Output Format**:
   - Set "action" with tool name and input when using a tool (set "answer" to empty string)
   - Set "answer" to a JSON string (escaped properly) when providing final structured output (set "action" to null)
   - Set "done" to "true" only when task is complete

### Example Responses

**When using a tool**:
{
  "thoughts": [{"content": "I need to search for information"}],
  "action": {
    "name": "search",
    "input": {"query": "search term"}
  },
  "answer": "",
  "done": false
}

**When providing final answer**:
{
  "thoughts": [{"content": "I have gathered all information"}],
  "action": null,
  "answer": "{\"your_field\": \"value\", \"another_field\": 123}",
  "done": true
}

### Important Notes

- The "answer" field must contain a properly escaped JSON string that can be unmarshaled into the target structure matching the provided schema
- Never include markdown formatting like triple backticks around your response
- Never include explanatory text before or after the JSON
- Ensure all JSON is valid and properly formatted
`, a.config.Output.OutputSchema)

	// Replace existing answer format section or append
	if strings.Contains(prompt, "## Response Format") {
		// Find and replace the answer format section
		parts := strings.Split(prompt, "## Response Format")
		if len(parts) >= 2 {
			return parts[0] + "## Response Format" + instructions
		}
	}

	return prompt + instructions
}

// RegisterTool registers a tool with the agent.
//
// The tool must be a *tools.Tool type.
func (a *ReActAgent) RegisterTool(tool interface{}) error {
	if toolPtr, ok := tool.(*tools.Tool); ok {
		return a.tools.RegisterTool(toolPtr)
	}
	return fmt.Errorf("tool must be *tools.Tool type")
}

// UnregisterTool removes a tool from the agent by name.
func (a *ReActAgent) UnregisterTool(name string) error {
	return a.tools.UnregisterTool(name)
}

// Register is an alias for RegisterTool.
func (a *ReActAgent) Register(tool *tools.Tool) error {
	return a.tools.RegisterTool(tool)
}

// Get retrieves a tool from the agent by name.
func (a *ReActAgent) Get(name string) (interface{}, error) {
	return a.tools.Get(name)
}

// Execute runs a tool with the given input parameters.
func (a *ReActAgent) Execute(name string, input map[string]interface{}) (string, error) {
	return a.tools.Execute(name, input)
}

// List returns the names of all registered tools.
func (a *ReActAgent) List() []string {
	return a.tools.List()
}

// Run executes the agent with the given query and returns the response.
//
// It uses the default timeout from config.
func (a *ReActAgent) Run(ctx context.Context, query string) (*ReActResponse, error) {
	return a.RunWithCallback(ctx, query, nil)
}

// RunWithCallback executes the agent with a callback function for each step.
//
// The callback is invoked after each action execution with the step details.
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

	if a.config.Debug {
		a.logger.Debug("[DEBUG] Initial user query", map[string]interface{}{
			"query": query,
		})
	}

	steps := make([]Step, 0)
	thoughts := make([]Thought, 0)

	var timeoutCtx context.Context
	var cancel context.CancelFunc

	if a.config.Timeout > 0 {
		timeoutCtx, cancel = context.WithTimeout(ctx, a.config.Timeout)
		defer cancel()
	} else {
		timeoutCtx = ctx
	}

	for iteration := 0; iteration < a.config.MaxIterations; iteration++ {
		if a.config.Timeout > 0 {
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
		}

		a.logger.Debug("Iteration start", map[string]interface{}{
			"iteration": iteration + 1,
		})

		systemPrompt := a.GetSystemPrompt()
		if a.config.Debug {
			a.logger.Debug("[DEBUG] System Prompt", map[string]interface{}{
				"prompt_length":  len(systemPrompt),
				"prompt_preview": systemPrompt[:min(500, len(systemPrompt))] + "...",
			})
		}

		if a.config.Debug && len(messages) > 0 {
			a.logger.Debug("[DEBUG] Message History Before LLM Call", map[string]interface{}{
				"message_count": len(messages),
			})
			for i, msg := range messages {
				a.logger.Debug("[DEBUG] Message", map[string]interface{}{
					"index":           i,
					"role":            string(msg.Role),
					"length":          len(msg.Content),
					"content_preview": msg.Content[:min(200, len(msg.Content))] + "...",
				})
			}
		}

		response, err := a.llm.GenerateWithSystem(systemPrompt, messages)
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

		if a.config.Debug {
			a.logger.Debug("[DEBUG] LLM Response", map[string]interface{}{
				"raw_response": response,
				"parsed":       parsed,
			})
			for i, thought := range parsed.Thoughts {
				a.logger.Debug("[DEBUG] Thought", map[string]interface{}{
					"index":   i + 1,
					"content": thought.Content,
				})
			}
			if parsed.Action != nil {
				a.logger.Debug("[DEBUG] Action", map[string]interface{}{
					"tool":  parsed.Action.Name,
					"input": parsed.Action.Input,
				})
			}
			if parsed.Answer != "" {
				a.logger.Debug("[DEBUG] Answer", map[string]interface{}{
					"answer": parsed.Answer,
				})
			}
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
			// Special handling for use_skill action
			if parsed.Action.Name == "use_skill" {
				skillName, _ := parsed.Action.Input["skill_name"].(string)
				if skillName == "" {
					skillName, _ = parsed.Action.Input["name"].(string)
				}

				latestThought := parsed.Thoughts[len(parsed.Thoughts)-1]
				step := &Step{
					Thought: &latestThought,
					Action:  parsed.Action,
				}

				a.logger.Info("Using skill", map[string]interface{}{
					"skill": skillName,
				})

				// Handle skill usage
				skillResponse, err := a.handleSkillUsage(timeoutCtx, skillName, query)
				if err != nil {
					step.Error = err.Error()
					step.Observation = &Observation{Content: fmt.Sprintf("Error: %v", err)}
				} else {
					step.Observation = &Observation{Content: skillResponse}
				}

				steps = append(steps, *step)
				if callback != nil {
					callback(step)
				}

				// Add skill response to message history
				messages = append(messages, llm.Message{
					Role: llm.RoleAssistant,
					Content: fmt.Sprintf("{\"thoughts\": [{\"content\": %s}], \"action\": {\"name\": \"use_skill\", \"input\": {\"skill_name\": \"%s\"}}, \"observation\": %s}",
						latestThought.Content, skillName, step.Observation.Content),
				})

				// After using skill, agent should provide final answer
				continue
			}

			// Regular tool execution
			latestThought := parsed.Thoughts[len(parsed.Thoughts)-1]
			step := &Step{
				Thought: &latestThought,
				Action:  parsed.Action,
			}

			a.logger.Info("Executing action", map[string]interface{}{
				"tool":  parsed.Action.Name,
				"input": parsed.Action.Input,
			})

			if a.config.Debug {
				inputJSON, _ := json.MarshalIndent(parsed.Action.Input, "", "  ")
				a.logger.Debug("[DEBUG] Executing tool", map[string]interface{}{
					"tool":  parsed.Action.Name,
					"input": parsed.Action.Input,
				})
				a.logger.Debug("[DEBUG] Tool input (JSON)", map[string]interface{}{
					"tool":  parsed.Action.Name,
					"input": string(inputJSON),
				})
			}

			result, err := a.tools.Execute(parsed.Action.Name, parsed.Action.Input)
			if err != nil {
				step.Error = err.Error()
				a.logger.Error("Tool execution failed", map[string]interface{}{
					"tool":  parsed.Action.Name,
					"error": err.Error(),
				})
				if a.config.Debug {
					a.logger.Debug("[DEBUG] Tool execution failed", map[string]interface{}{
						"tool":  parsed.Action.Name,
						"error": err.Error(),
					})
				}
			} else {
				step.Observation = &Observation{Content: result}
				a.logger.Debug("Tool execution succeeded", map[string]interface{}{
					"tool":   parsed.Action.Name,
					"result": result,
				})
				if a.config.Debug {
					a.logger.Debug("[DEBUG] Tool execution succeeded", map[string]interface{}{
						"tool":       parsed.Action.Name,
						"result":     result,
						"result_len": len(result),
					})
				}
			}

			steps = append(steps, *step)

			if a.config.Debug {
				a.logger.Debug("[DEBUG] Adding agent action to message history", map[string]interface{}{
					"thoughts_count": len(parsed.Thoughts),
					"action":         fmt.Sprintf("%s(%v)", parsed.Action.Name, parsed.Action.Input),
				})
			}

			messages = append(messages, llm.Message{
				Role:    llm.RoleAssistant,
				Content: response,
			})

			observationMsg := fmt.Sprintf("Observation: %s", result)
			if err != nil {
				observationMsg = fmt.Sprintf("Observation: Error - %s", err.Error())
			}

			if a.config.Debug {
				a.logger.Debug("[DEBUG] Adding observation to messages", map[string]interface{}{
					"observation":   observationMsg,
					"message_count": len(messages) + 1,
				})
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

// parseResponse parses the LLM response string into a ReActResponse.
func (a *ReActAgent) parseResponse(response string) (*ReActResponse, error) {
	return a.parser.Parse(response)
}

// RunStructured runs the agent and returns a struct-based response.
//
// The generic type T specifies the output structure. The agent will generate
// JSON that matches the structure of T.
func RunStructured[T any](agent *ReActAgent, ctx context.Context, query string) (*StructuredResponse[T], error) {
	return RunStructuredWithCallback[T](agent, ctx, query, nil)
}

// RunStructuredWithCallback runs the agent with structured output and callback support.
//
// The callback is invoked for each step during execution.
func RunStructuredWithCallback[T any](agent *ReActAgent, ctx context.Context, query string, callback func(step *Step)) (*StructuredResponse[T], error) {
	// Get the type of T
	var zeroT T
	outputType := reflect.TypeOf(zeroT)
	if outputType.Kind() == reflect.Ptr {
		outputType = outputType.Elem()
	}

	// Create struct parser and generate schema
	parser := NewStructParser()
	if agent.config.Output != nil && agent.config.Output.MaxNestingDepth > 0 {
		parser.SetMaxNestingDepth(agent.config.Output.MaxNestingDepth)
	}

	schema, err := parser.ParseStruct(outputType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse output struct type: %w", err)
	}

	// Store output schema in config for prompt injection
	if agent.config.Output == nil {
		agent.config.Output = &OutputConfig{}
	}
	agent.config.Output.EnableStructuredOutput = true
	agent.config.Output.OutputType = outputType
	agent.config.Output.OutputSchema = parser.ToJSONSchema(schema)

	// Run the agent with standard callback
	resp, err := agent.RunWithCallback(ctx, query, callback)
	if err != nil {
		return nil, err
	}

	// Parse the answer into the target struct
	result := &StructuredResponse[T]{
		ReActResponse: resp,
	}

	if resp.Done && resp.Answer != "" {
		var output T
		if err := json.Unmarshal([]byte(resp.Answer), &output); err != nil {
			return nil, fmt.Errorf("failed to parse structured answer: %w", err)
		}
		result.Output = &output
	}

	return result, nil
}

// Stream executes the agent with streaming LLM responses.
//
// The callback function is invoked for each chunk of the LLM response.
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

	var timeoutCtx context.Context
	var cancel context.CancelFunc

	if a.config.Timeout > 0 {
		timeoutCtx, cancel = context.WithTimeout(ctx, a.config.Timeout)
		defer cancel()
	} else {
		timeoutCtx = ctx
	}

	for iteration := 0; iteration < a.config.MaxIterations; iteration++ {
		if a.config.Timeout > 0 {
			select {
			case <-timeoutCtx.Done():
				return fmt.Errorf("agent timeout after %v", a.config.Timeout)
			default:
			}
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

// Close closes the LLM connection and logger.
func (a *ReActAgent) Close() error {
	if a.llm != nil {
		return a.llm.Close()
	}
	if a.logger != nil {
		return a.logger.Close()
	}
	return nil
}
