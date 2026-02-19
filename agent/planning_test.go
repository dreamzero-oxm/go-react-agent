package agent

import (
	"testing"

	"github.com/dreamzero-oxm/go-react-agent/logger"
)

func TestDefaultPlanConfig(t *testing.T) {
	config := DefaultPlanConfig()

	if config.Enabled {
		t.Error("Expected planning to be disabled by default for backward compatibility")
	}
	if !config.ReplanEnabled {
		t.Error("Expected re-planning to be enabled by default")
	}
	if config.ReplanEvery != 1 {
		t.Errorf("Expected ReplanEvery 1, got %d", config.ReplanEvery)
	}
}

func TestPlanStep(t *testing.T) {
	step := &PlanStep{
		ID:          "step-1",
		Description: "Test step",
		Tool:        "echo",
		Status:      "pending",
	}

	if step.ID != "step-1" {
		t.Errorf("Expected ID 'step-1', got %s", step.ID)
	}
	if step.Status != "pending" {
		t.Errorf("Expected status 'pending', got %s", step.Status)
	}
}

func TestPlan(t *testing.T) {
	plan := &Plan{
		Query: "Test query",
		Steps: []*PlanStep{
			{ID: "step-1", Description: "First step", Status: "pending"},
			{ID: "step-2", Description: "Second step", Status: "pending"},
		},
		Status:    "planning",
		Reasoning: "Test reasoning",
	}

	if plan.Query != "Test query" {
		t.Errorf("Expected query 'Test query', got %s", plan.Query)
	}
	if len(plan.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(plan.Steps))
	}
	if plan.Status != "planning" {
		t.Errorf("Expected status 'planning', got %s", plan.Status)
	}
}

func TestPlanResponse(t *testing.T) {
	resp := &PlanResponse{
		Reasoning: "Test reasoning",
		Steps: []*PlanStep{
			{ID: "step-1", Description: "First step", Status: "pending"},
		},
	}

	if resp.Reasoning != "Test reasoning" {
		t.Errorf("Expected reasoning 'Test reasoning', got %s", resp.Reasoning)
	}
	if len(resp.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(resp.Steps))
	}
}

func TestReActAgentWithPlanning_DefaultConfig(t *testing.T) {
	log := logger.NewMultiLogger()
	log.Disable()
	config := DefaultConfig()
	planConfig := DefaultPlanConfig()

	// PlanConfig should be included in Config
	if config.PlanConfig == nil {
		config.PlanConfig = planConfig
	}

	agent := NewReActAgentWithPlanning(nil, config, planConfig, log)

	if agent == nil {
		t.Error("Expected agent to be created")
	}

	if agent.planConfig == nil {
		t.Error("Expected planConfig to be set")
	}
}
