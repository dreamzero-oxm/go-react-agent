package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dreamzero-oxm/go-react-agent/agent"
	"github.com/dreamzero-oxm/go-react-agent/llm"
	"github.com/dreamzero-oxm/go-react-agent/logger"
	"github.com/dreamzero-oxm/go-react-agent/tools"
)

func main() {
	// Setup logging
	multiLog := logger.NewMultiLogger()
	multiLog.SetLevel(logger.LevelInfo)
	multiLog.AddConsoleLogger(true)

	// Configure LLM
	llmConfig := &llm.LLMConfig{
		APIKey:      os.Getenv("OPENAI_API_KEY"),
		BaseURL:     "https://api.openai.com/v1/chat/completions",
		Model:       "gpt-5.2",
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	openaiLLM, err := llm.NewOpenAILLM(llmConfig)
	if err != nil {
		panic(err)
	}
	defer openaiLLM.Close()

	// Create agent with planning enabled
	planConfig := agent.DefaultPlanConfig()
	planConfig.Enabled = true
	planConfig.ReplanEnabled = true

	config := agent.DefaultConfig()
	config.PlanConfig = planConfig

	planningAgent := agent.NewReActAgentWithPlanning(openaiLLM, config, planConfig, multiLog)
	planningAgent.InitializePlanning(openaiLLM)

	// Register tools
	tools.RegisterBuiltinToolsTo(planningAgent)

	// Run with planning
	ctx := context.Background()
	query := "Check the current weather in Chaozhou, China, and create a travel itinerary that includes the seaside"

	fmt.Printf("Query: %s\n\n", query)

	response, plan, err := planningAgent.RunWithPlan(ctx, query)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Initial Plan:\n")
	printPlan(plan)

	fmt.Printf("\nFinal Plan:\n")
	printPlan(planningAgent.GetPlan())

	fmt.Printf("\nFinal Answer: %s\n", response.Answer)
}

func printPlan(plan *agent.Plan) {
	if plan == nil {
		fmt.Println("No plan available")
		return
	}
	fmt.Printf("Status: %s\n", plan.Status)
	fmt.Printf("Reasoning: %s\n\n", plan.Reasoning)
	for i, step := range plan.Steps {
		fmt.Printf("%d. [%s] %s\n", i+1, step.Status, step.Description)
		if step.Tool != "" {
			fmt.Printf("   Tool: %s\n", step.Tool)
		}
		if step.Result != "" {
			fmt.Printf("   Result: %s\n", step.Result)
		}
	}
}
