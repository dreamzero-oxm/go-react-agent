package main

import (
	"context"
	"fmt"

	// "os"

	"github.com/dreamzero-oxm/go-react-agent/agent"
	"github.com/dreamzero-oxm/go-react-agent/llm"
	"github.com/dreamzero-oxm/go-react-agent/logger"
	"github.com/dreamzero-oxm/go-react-agent/mcp"
	"github.com/dreamzero-oxm/go-react-agent/tools"
)

func main() {
	log := logger.NewMultiLogger()
	log.SetLevel(logger.LevelInfo)
	log.AddConsoleLogger(true)

	// llmConfig := &llm.LLMConfig{
	// 	APIKey:      os.Getenv("OPENAI_API_KEY"),
	// 	BaseURL:     "https://api.openai.com/v1/chat/completions",
	// 	Model:       "gpt-3.5-turbo",
	// 	Temperature: 0.7,
	// 	MaxTokens:   2000,
	// }
	llmConfig := &llm.LLMConfig{
		APIKey:      "YOUR-API-KEY",
		BaseURL:     "https://open.bigmodel.cn/api/coding/paas/v4",
		Model:       "glm-4.7",
		Temperature: 1,
		MaxTokens:   2000,
	}

	openaiLLM, err := llm.NewOpenAILLM(llmConfig)
	if err != nil {
		panic(err)
	}
	defer openaiLLM.Close()

	fmt.Println("=== Example 1: Agent without MCP ===")
	config := agent.DefaultConfig()
	reactAgent := agent.NewReActAgent(openaiLLM, config, log)
	tools.RegisterBuiltinToolsTo(reactAgent)

	ctx := context.Background()
	response, err := reactAgent.Run(ctx, "What is 15 * 7?")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Answer: %s\n", response.Answer)

	fmt.Println("\n=== Example 2: Agent with MCP Integration ===")
	mcpConfig := agent.DefaultConfig()
	mcpConfig.MCPConfig.Enabled = true
	mcpConfig.MCPConfig.AutoLoadConfig = true
	mcpConfig.Debug = true

	if mcpConfig.Debug {
		log.SetLevel(logger.LevelDebug)
	}

	mcpAgent, err := agent.NewAgentWithMCP(openaiLLM, mcpConfig, log)
	if err != nil {
		panic(err)
	}
	defer mcpAgent.Close()

	tools.RegisterBuiltinToolsTo(mcpAgent)

	response, err = mcpAgent.Run(ctx, "Use a tool to calculate 15 * 7")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Answer: %s\n", response.Answer)

	fmt.Println("\n=== Example 3: Check MCP Status ===")
	statuses, err := agent.GetMCPStatus()
	if err != nil {
		fmt.Printf("Failed to get MCP status: %v\n", err)
	} else {
		for _, status := range statuses {
			fmt.Printf("Server: %s, Status: %s\n", status.Name, status.Status)
			fmt.Printf("  Tools: %d\n", len(status.Tools))
			for _, tool := range status.Tools {
				fmt.Printf("    - %s: %s\n", tool.Name, tool.Description)
			}
		}
	}

	fmt.Println("\n=== Example 4: Manual MCP Manager ===")
	manualConfig := agent.DefaultConfig()
	manualAgent := agent.NewReActAgent(openaiLLM, manualConfig, log)

	mcpCfg, err := mcp.LoadConfig()
	if err != nil {
		fmt.Printf("No MCP config found: %v\n", err)
	} else {
		mcpManager := mcp.NewManager(mcpCfg)
		if err := mcpManager.Start(); err != nil {
			fmt.Printf("Failed to start MCP manager: %v\n", err)
		} else {
			defer mcpManager.Stop()

			if err := manualAgent.WithMCPManager(mcpManager); err != nil {
				fmt.Printf("Failed to register MCP tools: %v\n", err)
			} else {
				fmt.Printf("Manual MCP tools registered\n")
			}
		}
	}
}
