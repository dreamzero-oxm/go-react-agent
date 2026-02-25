// Package main demonstrates the usage of Claude Code Skills with go-react-agent.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dreamzero-oxm/go-react-agent/agent"
	"github.com/dreamzero-oxm/go-react-agent/llm"
	"github.com/dreamzero-oxm/go-react-agent/logger"
)

func main() {
	log := logger.NewMultiLogger()
	log.SetLevel(logger.LevelInfo)
	log.AddConsoleLogger(true)

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
		panic(err)
	}
	defer openaiLLM.Close()

	// Create agent with Claude Code Skills enabled
	config := agent.DefaultConfig()
	config.SkillConfig.Enabled = true
	config.SkillConfig.AutoLoadSkills = true
	config.SkillConfig.MaxSkillsPerQuery = 3

	skillAgent, err := agent.NewAgentWithSkills(openaiLLM, config, log)
	if err != nil {
		panic(err)
	}
	defer skillAgent.Close()

	fmt.Println("=== Claude Code Skills Example ===")

	// Check if skills are loaded
	if skillAgent.IsSkillEnabled() {
		fmt.Println("Claude Code Skills are enabled!")
		fmt.Printf("Loaded %d skills:\n", len(skillAgent.GetSkills()))

		for _, skill := range skillAgent.GetSkills() {
			fmt.Printf("  - %s: %s\n", skill.Name, skill.Description)
		}
	}

	// Query using skills
	ctx := context.Background()

	fmt.Println("\n--- Example 1: Go Programming Question ---")
	response, err := skillAgent.Run(ctx, "How do I properly handle errors in Go? What are some common mistakes to avoid?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Answer: %s\n", response.Answer)
	}

	fmt.Println("\n--- Example 2: API Design Question ---")
	response, err = skillAgent.Run(ctx, "What HTTP status code should I return when a user tries to create a resource that already exists?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Answer: %s\n", response.Answer)
	}

	fmt.Println("\n--- Example 3: Combined Knowledge ---")
	response, err = skillAgent.Run(ctx, "I'm building a REST API in Go. What's the best way to structure my error handling for API endpoints?")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Answer: %s\n", response.Answer)
	}
}
