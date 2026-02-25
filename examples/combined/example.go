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
		BaseURL:     "https://open.bigmodel.cn/api/coding/paas/v4",
		Model:       "glm-4.7",
		Temperature: 0.7,
		MaxTokens:   3000,
	}

	openaiLLM, err := llm.NewOpenAILLM(llmConfig)
	if err != nil {
		panic(err)
	}
	defer openaiLLM.Close()

	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║   Agent with Skills + MCP Integration Example             ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝\n")

	config := agent.DefaultConfig()

	config.SkillConfig.Enabled = true
	config.SkillConfig.AutoLoadSkills = true
	config.SkillConfig.MaxSkillsPerQuery = 5
	config.SkillConfig.GlobalSkillsDir = "~/.go-react-agent/skills/"
	config.SkillConfig.ProjectSkillsDir = "../claude-skills/"

	config.MCPConfig.Enabled = true
	config.MCPConfig.AutoLoadConfig = true

	config.Debug = true
	if config.Debug {
		log.SetLevel(logger.LevelDebug)
	}

	combinedAgent, err := agent.NewAgentWithMCP(openaiLLM, config, log)
	if err != nil {
		panic(err)
	}
	defer combinedAgent.Close()

	if err := combinedAgent.WithSkillIntegration(); err != nil {
		log.Error("Failed to load skills", map[string]interface{}{
			"error": err,
		})
	} else {
		log.Info("Skills loaded successfully", map[string]interface{}{
			"count": len(combinedAgent.GetSkills()),
		})
	}

	tools.RegisterBuiltinToolsTo(combinedAgent)

	ctx := context.Background()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Query 1: Using Skills (Go programming knowledge)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	query1 := "Explain the best practices for error handling in Go, including when to use errors.Is() and errors.As()"
	fmt.Printf("Query: %s\n\n", query1)

	response1, err := combinedAgent.Run(ctx, query1)
	if err != nil {
		panic(err)
	}

	fmt.Printf("\nFinal Answer:\n%s\n", response1.Answer)

	fmt.Println("\n\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Query 2: Using MCP Tools (calculation and file operations)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	query2 := "Calculate sum of all prime numbers between 1 and 50, then save the result to a file called prime_sum.txt"
	fmt.Printf("Query: %s\n\n", query2)

	response2, err := combinedAgent.Run(ctx, query2)
	if err != nil {
		panic(err)
	}

	fmt.Printf("\nFinal Answer:\n%s\n", response2.Answer)

	fmt.Println("\n\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Query 3: Combined Skills + MCP (complex task)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	query3 := "I'm building a Go web API. Use Go expert knowledge to design a REST endpoint structure, then calculate how many HTTP status codes exist and list them in a file"
	fmt.Printf("Query: %s\n\n", query3)

	response3, err := combinedAgent.Run(ctx, query3)
	if err != nil {
		panic(err)
	}

	fmt.Printf("\nFinal Answer:\n%s\n", response3.Answer)

	fmt.Println("\n\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Summary")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	skills := combinedAgent.GetSkills()
	fmt.Printf("\n📚 Loaded Skills: %d\n", len(skills))
	for _, skill := range skills {
		fmt.Printf("   • %s", skill.Name)
		if skill.Version != "" {
			fmt.Printf(" (v%s)", skill.Version)
		}
		fmt.Printf(" - %s\n", skill.Description)
	}

	mcpStatuses, err := agent.GetMCPStatus()
	if err == nil && len(mcpStatuses) > 0 {
		fmt.Printf("\n🔌 MCP Servers: %d\n", len(mcpStatuses))
		for _, status := range mcpStatuses {
			fmt.Printf("   • %s: %s (%d tools)\n", status.Name, status.Status, len(status.Tools))
		}
	} else {
		fmt.Printf("\n🔌 MCP Servers: No MCP servers configured\n")
	}

	fmt.Println("\n╔════════════════════════════════════════════════════════╗")
	fmt.Println("║   All queries completed successfully!                       ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
}
