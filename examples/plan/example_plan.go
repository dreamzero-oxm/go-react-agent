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
	multiLog := logger.NewMultiLogger()
	multiLog.SetLevel(logger.LevelInfo)
	multiLog.AddConsoleLogger(true)

	fileLog, err := multiLog.AddFileLogger("agent.log")
	if err != nil {
		fmt.Printf("Failed to add file logger: %v\n", err)
	} else {
		defer fileLog.Close()
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Please set OPENAI_API_KEY environment variable")
		os.Exit(1)
	}

	llmConfig := &llm.LLMConfig{
		APIKey:      apiKey,
		BaseURL:     "https://open.bigmodel.cn/api/coding/paas/v4",
		Model:       "glm-4.7",
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	openaiLLM, err := llm.NewOpenAILLM(llmConfig)
	if err != nil {
		fmt.Printf("Failed to create OpenAI LLM: %v\n", err)
		os.Exit(1)
	}
	defer openaiLLM.Close()

	planConfig := agent.DefaultPlanConfig()
	planConfig.Enabled = true
	planConfig.ReplanEnabled = true

	config := agent.DefaultConfig()
	config.PlanConfig = planConfig

	planningAgent := agent.NewReActAgentWithPlanning(openaiLLM, config, planConfig, multiLog)
	planningAgent.InitializePlanning(openaiLLM)

	tools.RegisterBuiltinToolsTo(planningAgent)

	ctx := context.Background()
	query := "Check current weather in Chaozhou, China, and create a travel itinerary that includes seaside"

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
