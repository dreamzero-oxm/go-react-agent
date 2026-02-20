package agent

// PlanStep represents a single step in the execution plan.
type PlanStep struct {
	// ID is the unique identifier for this step
	ID string `json:"id"`
	// Description describes what this step accomplishes
	Description string `json:"description"`
	// Tool is the name of the tool to use (empty if LLM should decide)
	Tool string `json:"tool,omitempty"`
	// Input contains the parameters for the tool
	Input map[string]interface{} `json:"input,omitempty"`
	// Status is the execution status (pending, in_progress, completed, failed)
	Status string `json:"status"`
	// Result contains the output from executing this step
	Result string `json:"result,omitempty"`
}

// Plan represents the overall execution plan for a query.
type Plan struct {
	// Query is the original user query
	Query string `json:"query"`
	// Steps is the list of steps in the execution plan
	Steps []*PlanStep `json:"steps"`
	// CurrentStep is the index of the currently executing step
	CurrentStep int `json:"current_step"`
	// Status is the plan status (planning, executing, completed, failed)
	Status string `json:"status"`
	// Reasoning explains the plan's approach and strategy
	Reasoning string `json:"reasoning,omitempty"`
}

// PlanResponse is the LLM response for plan generation.
type PlanResponse struct {
	// Reasoning explains the plan's approach and strategy
	Reasoning string `json:"reasoning"`
	// Steps is the list of steps in the generated plan
	Steps []*PlanStep `json:"steps"`
}

// PlanConfig holds configuration for planning behavior.
type PlanConfig struct {
	// Enabled enables the planning feature
	Enabled bool `json:"enabled"`
	// ReplanEnabled enables adaptive re-planning during execution
	ReplanEnabled bool `json:"replan_enabled"`
	// ReplanEvery specifies how often to re-plan (1 = after each step)
	ReplanEvery int `json:"replan_every"`
	// SystemPrompt is a custom planning system prompt (optional)
	SystemPrompt string `json:"-"`
}

// DefaultPlanConfig returns default planning configuration.
//
// Defaults:
//   - Enabled: false (opt-in for backward compatibility)
//   - ReplanEnabled: true
//   - ReplanEvery: 1
func DefaultPlanConfig() *PlanConfig {
	return &PlanConfig{
		Enabled:       false, // Opt-in for backward compatibility
		ReplanEnabled: true,
		ReplanEvery:   1,
	}
}
