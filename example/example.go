package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-react-agent/agent"
	"github.com/go-react-agent/llm"
	"github.com/go-react-agent/logger"
	"github.com/go-react-agent/tools"
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
		BaseURL:     "https://api.openai.com/v1/chat/completions",
		Model:       "gpt-3.5-turbo",
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	openaiLLM, err := llm.NewOpenAILLM(llmConfig)
	if err != nil {
		fmt.Printf("Failed to create OpenAI LLM: %v\n", err)
		os.Exit(1)
	}
	defer openaiLLM.Close()

	agentConfig := agent.DefaultConfig()
	agentConfig.MaxIterations = 10

	reactAgent := agent.NewReActAgent(openaiLLM, agentConfig, multiLog)

	customTool := &tools.Tool{
		Name:        "get_weather",
		Description: "Get current weather for a city",
		Parameters: map[string]tools.Parameter{
			"city": {
				Type:        "string",
				Description: "Name of the city",
				Required:    true,
			},
		},
		Execute: func(input map[string]interface{}) (string, error) {
			city, ok := input["city"].(string)
			if !ok {
				return "", fmt.Errorf("city must be a string")
			}
			return fmt.Sprintf("Weather in %s: Sunny, 25°C", city), nil
		},
	}

	if err := reactAgent.RegisterTool(customTool); err != nil {
		fmt.Printf("Failed to register tool: %v\n", err)
		os.Exit(1)
	}

	tools.RegisterBuiltinToolsTo(reactAgent)

	ctx := context.Background()

	query := "What's the weather in Tokyo? Also, calculate 15 * 7 for me."

	fmt.Printf("Query: %s\n\n", query)

	response, err := reactAgent.RunWithCallback(ctx, query, func(step *agent.Step) {
		if step.Action != nil {
			fmt.Printf("Action: %s\n", step.Action.Name)
			fmt.Printf("  Input: %v\n", step.Action.Input)
		}
		if step.Observation != nil {
			fmt.Printf("Observation: %s\n", step.Observation.Content)
		}
		if step.Error != "" {
			fmt.Printf("Error: %s\n", step.Error)
		}
		fmt.Println()
	})

	if err != nil {
		fmt.Printf("Agent failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nFinal Answer: %s\n", response.Answer)

	if err := multiLog.Close(); err != nil {
		fmt.Printf("Error closing logger: %v\n", err)
	}
}
