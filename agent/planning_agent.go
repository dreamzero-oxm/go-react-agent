package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dreamzero-oxm/go-react-agent/llm"
	"github.com/dreamzero-oxm/go-react-agent/logger"
)

// PlanningAgent handles plan generation and re-planning
type PlanningAgent struct {
	llm    llm.LLM
	tools  ToolRegistry
	logger logger.Logger
	config *PlanConfig
}

// NewPlanningAgent creates a new planning agent
func NewPlanningAgent(llm llm.LLM, tools ToolRegistry, config *PlanConfig, log logger.Logger) *PlanningAgent {
	return &PlanningAgent{
		llm:    llm,
		tools:  tools,
		logger: log,
		config: config,
	}
}

// CreateInitialPlan generates an initial plan for the given query
func (p *PlanningAgent) CreateInitialPlan(ctx context.Context, query string) (*Plan, error) {
	p.logger.Info("Creating initial plan", map[string]interface{}{"query": query})

	systemPrompt := p.getSystemPrompt()

	messages := []llm.Message{
		{Role: llm.RoleUser, Content: fmt.Sprintf("Create an execution plan for: %s", query)},
	}

	response, err := p.llm.GenerateWithSystem(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("plan generation failed: %w", err)
	}

	planResp, err := p.parsePlanResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plan: %w", err)
	}

	plan := &Plan{
		Query:     query,
		Steps:     planResp.Steps,
		Status:    "planning",
		Reasoning: planResp.Reasoning,
	}

	// Initialize step IDs and statuses
	for i, step := range plan.Steps {
		if step.ID == "" {
			step.ID = fmt.Sprintf("step-%d", i+1)
		}
		if step.Status == "" {
			step.Status = "pending"
		}
	}

	p.logger.Info("Initial plan created", map[string]interface{}{"steps": len(plan.Steps)})
	return plan, nil
}

// Replan updates the plan based on current execution state
func (p *PlanningAgent) Replan(ctx context.Context, plan *Plan, lastStep *PlanStep, observation string) (*Plan, error) {
	p.logger.Info("Re-planning", map[string]interface{}{"completed_step": lastStep.ID})

	lastStep.Status = "completed"
	lastStep.Result = observation

	systemPrompt := p.getReplanSystemPrompt()

	messages := []llm.Message{
		{Role: llm.RoleUser, Content: p.formatReplanRequest(plan, lastStep, observation)},
	}

	response, err := p.llm.GenerateWithSystem(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("re-planning failed: %w", err)
	}

	planResp, err := p.parsePlanResponse(response)
	if err != nil {
		return p.updatePlanStatus(plan, "failed"), fmt.Errorf("failed to parse replan: %w", err)
	}

	return p.mergePlans(plan, planResp), nil
}

func (p *PlanningAgent) getSystemPrompt() string {
	if p.config.SystemPrompt != "" {
		return p.injectTools(p.config.SystemPrompt)
	}
	return p.injectTools(`You are an expert planning agent. Your task is to analyze complex queries and break them down into clear, actionable execution plans.

## Planning Process

1. **Analyze the Query**: Understand what the user wants to accomplish
2. **Identify Required Actions**: Determine what steps are necessary
3. **Select Appropriate Tools**: Choose the best tools for each step
4. **Order Steps Logically**: Arrange steps in a logical sequence
5. **Consider Dependencies**: Identify which steps depend on others

## Response Format

Respond ONLY with valid JSON in this exact format:

{
  "reasoning": "Explain your overall approach, why you chose these steps, and how they work together to accomplish the goal",
  "steps": [
    {
      "id": "step-1",
      "description": "Clear, specific description of what this step accomplishes",
      "tool": "tool_name (if a specific tool should be used, omit if LLM should decide)",
      "input": {},
      "status": "pending"
    }
  ]
}

## Planning Best Practices

1. **Be Specific**: Each step should have a clear, specific purpose
2. **Be Atomic**: Steps should be small enough to complete independently
3. **Be Sequential**: Steps should follow a logical order
4. **Use Tools Appropriately**: Only specify a tool if it's clearly the right choice
5. **Handle Uncertainty**: If you're unsure about a step, omit the tool field to let the execution agent decide
6. **Consider Edge Cases**: Think about what could go wrong and plan accordingly

## Available Tools

{{tools}}

## Examples

**Query**: "Calculate 15 * 7, then check if the result is even"

**Good Plan**:
{
  "reasoning": "I need to first calculate the product, then analyze the result. The calculate tool can handle the multiplication, and I can determine if the result is even from the calculation output.",
  "steps": [
    {
      "id": "step-1",
      "description": "Calculate 15 multiplied by 7",
      "tool": "calculate",
      "input": {"expression": "15 * 7"},
      "status": "pending"
    },
    {
      "id": "step-2",
      "description": "Check if the result is even (divisible by 2)",
      "status": "pending"
    }
  ]
}

**Query**: "Read the file config.json and update the port to 8080"

**Good Plan**:
{
  "reasoning": "I need to first read the current file contents to understand the structure, then write the updated content back. This requires sequential execution since I need the file contents first.",
  "steps": [
    {
      "id": "step-1",
      "description": "Read config.json to see current contents",
      "tool": "read_file",
      "input": {"path": "config.json"},
      "status": "pending"
    },
    {
      "id": "step-2",
      "description": "Update the port field to 8080 and write back to file",
      "tool": "write_file",
      "status": "pending"
    }
  ]
}

## Important Rules

1. Response must be valid JSON only - no additional text
2. Each step must have a unique id (step-1, step-2, etc.)
3. The "reasoning" field should explain your thought process
4. If a step doesn't require a specific tool, omit the "tool" field
5. Keep descriptions clear and actionable
6. Consider what information from previous steps might be needed`)
}

func (p *PlanningAgent) getReplanSystemPrompt() string {
	return p.injectTools(`You are an expert re-planning agent. Your task is to update execution plans based on progress and new information.

## Re-planning Process

1. **Review Progress**: Examine completed steps and their results
2. **Assess Current State**: Determine what has been accomplished
3. **Validate Original Plan**: Check if the remaining plan is still appropriate
4. **Adjust as Needed**: Add, remove, or modify remaining steps
5. **Handle Failures**: Create recovery strategies for failed steps

## Response Format

Respond ONLY with valid JSON in this exact format:

{
  "reasoning": "Explain what changed, why you're updating the plan, and how the new plan addresses the current situation",
  "steps": [
    {
      "id": "step-1",
      "description": "Step description",
      "tool": "tool_name (optional)",
      "input": {},
      "status": "pending"
    }
  ]
}

## Re-planning Best Practices

1. **Preserve Completed Work**: Keep all successfully completed steps
2. **Learn from Results**: Use actual results to inform next steps
3. **Be Flexible**: Don't hesitate to change the original plan
4. **Handle Errors Gracefully**: Create recovery plans for failures
5. **Optimize**: Remove unnecessary steps if the goal is already met

## Available Tools

{{tools}}

## Examples

**Scenario**: A step failed because a file doesn't exist

**Re-plan Response**:
{
  "reasoning": "The read_file step failed because the file doesn't exist. I should create the file first with default content, then retry the read operation, or proceed with creating the file directly if the goal is to create it.",
  "steps": [
    {
      "id": "step-1",
      "description": "Create config.json with default content",
      "tool": "write_file",
      "input": {"path": "config.json", "content": "{}"},
      "status": "pending"
    },
    {
      "id": "step-2",
      "description": "Update the port field to 8080",
      "tool": "write_file",
      "status": "pending"
    }
  ]
}

**Scenario**: A calculation returned an unexpected result

**Re-plan Response**:
{
  "reasoning": "The calculation returned 105, which is different than expected. I need to verify this result and adjust my next steps accordingly. The result is indeed odd (not even), so I should report this finding.",
  "steps": [
    {
      "id": "step-1",
      "description": "Verify the calculation result and report findings",
      "status": "pending"
    }
  ]
}

## Important Rules

1. Always preserve completed steps in your response
2. Update step IDs to maintain a logical sequence
3. Clearly explain why changes are being made
4. Be pragmatic - focus on completing the goal efficiently
5. Consider alternative approaches when original plans fail`)
}

func (p *PlanningAgent) injectTools(prompt string) string {
	toolsSchema := p.tools.GetToolsSchema()
	if strings.Contains(prompt, "{{tools}}") {
		return strings.ReplaceAll(prompt, "{{tools}}", toolsSchema)
	}
	if strings.Contains(prompt, "{{TOOLS}}") {
		return strings.ReplaceAll(prompt, "{{TOOLS}}", toolsSchema)
	}
	return fmt.Sprintf("%s\n\n## Available Tools\n\n%s", prompt, toolsSchema)
}

func (p *PlanningAgent) parsePlanResponse(response string) (*PlanResponse, error) {
	cleaned := strings.TrimSpace(response)
	// Remove markdown code blocks if present
	if strings.HasPrefix(cleaned, "```json") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
		cleaned = strings.TrimSuffix(cleaned, "```")
		cleaned = strings.TrimSpace(cleaned)
	} else if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```")
		cleaned = strings.TrimSuffix(cleaned, "```")
		cleaned = strings.TrimSpace(cleaned)
	}

	var result PlanResponse
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	if len(result.Steps) == 0 {
		return nil, fmt.Errorf("plan must contain at least one step")
	}

	return &result, nil
}

func (p *PlanningAgent) formatReplanRequest(plan *Plan, lastStep *PlanStep, observation string) string {
	return fmt.Sprintf(`Update the execution plan based on completed step.

Original Query: %s

Completed Step:
- ID: %s
- Description: %s
- Result: %s

Current Plan:
%s

Please provide an updated plan.`, plan.Query, lastStep.ID, lastStep.Description, observation, p.formatPlan(plan))
}

func (p *PlanningAgent) formatPlan(plan *Plan) string {
	result := ""
	for _, step := range plan.Steps {
		status := step.Status
		if status == "" {
			status = "pending"
		}
		result += fmt.Sprintf("- [%s] %s: %s\n", status, step.ID, step.Description)
	}
	return result
}

func (p *PlanningAgent) mergePlans(original *Plan, newResp *PlanResponse) *Plan {
	// Keep completed steps, replace/update pending steps
	updated := &Plan{
		Query:     original.Query,
		Reasoning: newResp.Reasoning,
		Status:    "executing",
		Steps:     []*PlanStep{},
	}

	// Add completed steps from original
	for _, step := range original.Steps {
		if step.Status == "completed" {
			updated.Steps = append(updated.Steps, step)
		}
	}

	// Add new/updated steps from response
	for i, step := range newResp.Steps {
		if step.ID == "" {
			step.ID = fmt.Sprintf("step-%d", len(updated.Steps)+i+1)
		}
		if step.Status == "" {
			step.Status = "pending"
		}
		updated.Steps = append(updated.Steps, step)
	}

	return updated
}

func (p *PlanningAgent) updatePlanStatus(plan *Plan, status string) *Plan {
	plan.Status = status
	return plan
}
