package agent

// PlanStep represents a single step in the execution plan
type PlanStep struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	Tool        string                 `json:"tool,omitempty"`
	Input       map[string]interface{} `json:"input,omitempty"`
	Status      string                 `json:"status"` // pending, in_progress, completed, failed
	Result      string                 `json:"result,omitempty"`
}

// Plan represents the overall execution plan
type Plan struct {
	Query        string      `json:"query"`
	Steps        []*PlanStep `json:"steps"`
	CurrentStep  int         `json:"current_step"`
	Status       string      `json:"status"` // planning, executing, completed, failed
	Reasoning    string      `json:"reasoning,omitempty"`
}

// PlanResponse is the LLM response for plan generation
type PlanResponse struct {
	Reasoning string      `json:"reasoning"`
	Steps     []*PlanStep `json:"steps"`
}

// PlanConfig holds configuration for planning behavior
type PlanConfig struct {
	Enabled       bool   `json:"enabled"`        // Enable planning feature
	ReplanEnabled bool   `json:"replan_enabled"` // Enable adaptive re-planning
	ReplanEvery   int    `json:"replan_every"`   // Re-plan every N steps (1 = after each step)
	SystemPrompt  string `json:"-"`              // Custom planning system prompt
}

// DefaultPlanConfig returns default planning configuration
func DefaultPlanConfig() *PlanConfig {
	return &PlanConfig{
		Enabled:       false, // Opt-in for backward compatibility
		ReplanEnabled: true,
		ReplanEvery:   1,
	}
}
