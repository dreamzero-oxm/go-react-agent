package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/dreamzero-oxm/go-react-agent/llm"
	"github.com/dreamzero-oxm/go-react-agent/logger"
)

// ReActAgentWithPlanning extends ReActAgent with planning capabilities
type ReActAgentWithPlanning struct {
	*ReActAgent
	planAgent   *PlanningAgent
	planConfig  *PlanConfig
	currentPlan *Plan
}

// NewReActAgentWithPlanning creates a new agent with planning support
func NewReActAgentWithPlanning(llm llm.LLM, config *Config, planConfig *PlanConfig, log logger.Logger) *ReActAgentWithPlanning {
	baseAgent := NewReActAgent(llm, config, log)

	// Ensure PlanConfig is set
	if planConfig == nil {
		planConfig = DefaultPlanConfig()
	}

	return &ReActAgentWithPlanning{
		ReActAgent: baseAgent,
		planConfig: planConfig,
	}
}

// InitializePlanning sets up the planning agent with an LLM
func (a *ReActAgentWithPlanning) InitializePlanning(llm llm.LLM) {
	a.planAgent = NewPlanningAgent(llm, a.tools, a.planConfig, a.logger)
}

// RunWithPlan executes the query with planning enabled
func (a *ReActAgentWithPlanning) RunWithPlan(ctx context.Context, query string) (*ReActResponse, *Plan, error) {
	if !a.planConfig.Enabled {
		// Fallback to standard execution
		response, err := a.ReActAgent.Run(ctx, query)
		return response, nil, err
	}

	// Create initial plan
	plan, err := a.planAgent.CreateInitialPlan(ctx, query)
	if err != nil {
		a.logger.Warn("Plan creation failed, falling back to standard execution", map[string]interface{}{"error": err.Error()})
		response, err := a.ReActAgent.Run(ctx, query)
		return response, nil, err
	}

	a.currentPlan = plan
	response, err := a.executeWithPlan(ctx, plan, query)
	return response, plan, err
}

// Run implements Agent interface - delegates to standard Run or RunWithPlan
func (a *ReActAgentWithPlanning) Run(ctx context.Context, query string) (*ReActResponse, error) {
	if a.planConfig.Enabled {
		response, _, err := a.RunWithPlan(ctx, query)
		return response, err
	}
	return a.ReActAgent.Run(ctx, query)
}

// RunWithCallback implements Agent interface with callback support
func (a *ReActAgentWithPlanning) RunWithCallback(ctx context.Context, query string, callback func(step *Step)) (*ReActResponse, error) {
	if a.planConfig.Enabled {
		response, _, err := a.RunWithPlan(ctx, query)
		return response, err
	}
	return a.ReActAgent.RunWithCallback(ctx, query, callback)
}

// GetPlan returns the current execution plan
func (a *ReActAgentWithPlanning) GetPlan() *Plan {
	return a.currentPlan
}

// SetPlanConfig updates the planning configuration
func (a *ReActAgentWithPlanning) SetPlanConfig(config *PlanConfig) {
	a.planConfig = config
	if a.planAgent != nil {
		a.planAgent.config = config
	}
}

// RegisterTool registers a tool with the agent
func (a *ReActAgentWithPlanning) RegisterTool(tool interface{}) error {
	return a.ReActAgent.RegisterTool(tool)
}

// UnregisterTool unregisters a tool by name
func (a *ReActAgentWithPlanning) UnregisterTool(name string) error {
	return a.ReActAgent.UnregisterTool(name)
}

// Close closes the agent and releases resources
func (a *ReActAgentWithPlanning) Close() error {
	return a.ReActAgent.Close()
}

func (a *ReActAgentWithPlanning) executeWithPlan(ctx context.Context, plan *Plan, query string) (*ReActResponse, error) {
	plan.Status = "executing"

	messages := []llm.Message{
		{Role: llm.RoleUser, Content: query},
	}

	thoughts := []Thought{}

	for stepIndex := 0; stepIndex < len(plan.Steps) && stepIndex < a.config.MaxIterations; stepIndex++ {
		step := plan.Steps[stepIndex]

		if step.Status == "completed" {
			continue
		}

		step.Status = "in_progress"

		var result string
		var err error

		// Execute the step
		if step.Tool != "" {
			a.logger.Info("Executing planned step", map[string]interface{}{"step": step.ID, "tool": step.Tool})
			result, err = a.tools.Execute(step.Tool, step.Input)
		} else {
			// No tool specified - let LLM decide
			result, err = a.executeWithLLM(ctx, messages, step)
		}

		if err != nil {
			step.Status = "failed"
			step.Result = err.Error()

			// Re-plan on failure if enabled
			if a.planConfig.ReplanEnabled {
				updatedPlan, replanErr := a.planAgent.Replan(ctx, plan, step, err.Error())
				if replanErr != nil {
					a.logger.Warn("Re-planning failed", map[string]interface{}{"error": replanErr.Error()})
				} else {
					plan = updatedPlan
					a.currentPlan = plan
					// Find the first pending step in the updated plan
					nextStepIndex := a.findFirstPendingStep(plan)
					if nextStepIndex >= 0 {
						// Set to one before the target index (will be incremented by loop)
						stepIndex = nextStepIndex - 1
					} else {
						// All steps completed, exit loop
						stepIndex = len(plan.Steps)
					}
					continue
				}
			}

			return &ReActResponse{
				Thoughts: thoughts,
				Done:     false,
			}, fmt.Errorf("step %s failed: %w", step.ID, err)
		}

		step.Status = "completed"
		step.Result = result

		thoughts = append(thoughts, Thought{Content: fmt.Sprintf("Step %s completed: %s", step.ID, result)})

		// Re-plan after step if enabled
		if a.planConfig.ReplanEnabled && (stepIndex+1)%a.planConfig.ReplanEvery == 0 {
			updatedPlan, replanErr := a.planAgent.Replan(ctx, plan, step, result)
			if replanErr != nil {
				a.logger.Warn("Re-planning failed", map[string]interface{}{"error": replanErr.Error()})
			} else {
				plan = updatedPlan
				a.currentPlan = plan
			}
		}
	}

	plan.Status = "completed"

	// Get final answer from LLM
	finalPrompt := fmt.Sprintf("Plan executed successfully. Provide a final answer based on these results:\n%s", a.formatPlanResults(plan))
	messages = append(messages, llm.Message{Role: llm.RoleUser, Content: finalPrompt})

	response, err := a.llm.GenerateWithSystem(a.GetSystemPrompt(), messages)
	if err != nil {
		return &ReActResponse{
			Thoughts: thoughts,
			Answer:   fmt.Sprintf("All steps completed. %s", a.formatPlanResults(plan)),
			Done:     true,
		}, nil
	}

	if response == "" {
		return &ReActResponse{
			Thoughts: thoughts,
			Answer:   fmt.Sprintf("All steps completed. %s", a.formatPlanResults(plan)),
			Done:     true,
		}, nil
	}

	parsed, parseErr := a.parseResponse(response)
	if parsed == nil {
		return &ReActResponse{
			Thoughts: thoughts,
			Answer:   fmt.Sprintf("All steps completed. %s", a.formatPlanResults(plan)),
			Done:     true,
		}, parseErr
	}
	parsed.Thoughts = append(thoughts, parsed.Thoughts...)

	return parsed, nil
}

func (a *ReActAgentWithPlanning) executeWithLLM(ctx context.Context, messages []llm.Message, step *PlanStep) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	prompt := fmt.Sprintf("Execute this step: %s", step.Description)
	messages = append(messages, llm.Message{Role: llm.RoleUser, Content: prompt})

	response, err := a.llm.GenerateWithSystem(a.GetSystemPrompt(), messages)
	if err != nil {
		return "", err
	}

	parsed, err := a.parseResponse(response)
	if err != nil {
		return response, nil
	}

	if parsed == nil {
		return "Step completed", nil
	}

	if parsed.Action != nil {
		result, err := a.tools.Execute(parsed.Action.Name, parsed.Action.Input)
		if err != nil {
			return "", err
		}
		messages = append(messages, llm.Message{Role: llm.RoleUser, Content: fmt.Sprintf("Observation: %s", result)})
		return result, nil
	}

	if parsed.Answer != "" {
		return parsed.Answer, nil
	}

	return "Step completed", nil
}

func (a *ReActAgentWithPlanning) formatPlanResults(plan *Plan) string {
	var builder strings.Builder
	for _, step := range plan.Steps {
		if step.Status == "completed" {
			fmt.Fprintf(&builder, "- %s: %s\n", step.Description, step.Result)
		}
	}
	return builder.String()
}

// findFirstPendingStep finds the index of the first step with pending status
// Returns -1 if all steps are completed
func (a *ReActAgentWithPlanning) findFirstPendingStep(plan *Plan) int {
	for i, step := range plan.Steps {
		if step.Status == "pending" || step.Status == "failed" {
			return i
		}
	}
	return -1
}

// RunStructuredWithPlan executes the query with planning and structured output
func RunStructuredWithPlan[T any](agent *ReActAgentWithPlanning, ctx context.Context, query string) (*StructuredResponse[T], *Plan, error) {
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
		return nil, nil, fmt.Errorf("failed to parse output struct type: %w", err)
	}

	// Store output schema in config for prompt injection
	if agent.config.Output == nil {
		agent.config.Output = &OutputConfig{}
	}
	agent.config.Output.EnableStructuredOutput = true
	agent.config.Output.OutputType = outputType
	agent.config.Output.OutputSchema = parser.ToJSONSchema(schema)

	// Run with planning
	response, plan, err := agent.RunWithPlan(ctx, query)
	if err != nil {
		return nil, nil, err
	}

	// Parse the answer into the target struct
	result := &StructuredResponse[T]{
		ReActResponse: response,
	}

	if response.Done && response.Answer != "" {
		var output T
		if err := json.Unmarshal([]byte(response.Answer), &output); err != nil {
			return nil, nil, fmt.Errorf("failed to parse structured answer: %w", err)
		}
		result.Output = &output
	}

	return result, plan, nil
}
