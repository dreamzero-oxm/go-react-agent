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

type WeatherReport struct {
	City        string  `json:"city" agent:"desc:城市名称;required:true"`
	Temperature float64 `json:"temperature" agent:"desc:摄氏温度;required:true;range:-50,60"`
	Humidity    int     `json:"humidity" agent:"desc:相对湿度百分比;required:true;range:0,100"`
	Condition   string  `json:"condition" agent:"desc:天气状况;required:true;enum:sunny,cloudy,rainy,snowy,windy"`
	Description string  `json:"description" agent:"desc:天气详细描述"`
	HasAlert    bool    `json:"has_alert" agent:"desc:是否有天气预警"`
}

func main() {
	multiLog := logger.NewMultiLogger()
	multiLog.SetLevel(logger.LevelInfo)
	multiLog.AddConsoleLogger(true)

	fileLog, err := multiLog.AddFileLogger("agent_structured.log")
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
		fmt.Printf("Failed to create LLM: %v\n", err)
		os.Exit(1)
	}
	defer openaiLLM.Close()

	agentConfig := agent.DefaultConfig()
	agentConfig.MaxIterations = 10

	reactAgent := agent.NewReActAgent(openaiLLM, agentConfig, multiLog)

	if err := tools.RegisterBuiltinToolsTo(reactAgent); err != nil {
		fmt.Printf("Failed to register tools: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	query := "查询深圳当前的天气情况，包括温度、湿度和天气状况"

	fmt.Printf("╔══════════════════════════════════════════╗\n")
	fmt.Printf("║   React Agent 结构化输出示例           ║\n")
	fmt.Printf("╚══════════════════════════════════════════╝\n\n")
	fmt.Printf("Query: %s\n\n", query)

	response, err := agent.RunStructured[WeatherReport](reactAgent, ctx, query)
	if err != nil {
		fmt.Printf("Agent failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Thoughts:\n")
	for _, thought := range response.ReActResponse.Thoughts {
		fmt.Printf("  • %s\n", thought.Content)
	}

	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Structured Output:\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("  📍 城市: %s\n", response.Output.City)
	fmt.Printf("  🌡️  温度: %.1f°C\n", response.Output.Temperature)
	fmt.Printf("  💧 湿度: %d%%\n", response.Output.Humidity)

	var conditionIcon string
	switch response.Output.Condition {
	case "sunny":
		conditionIcon = "☀️"
	case "cloudy":
		conditionIcon = "☁️"
	case "rainy":
		conditionIcon = "🌧️"
	case "snowy":
		conditionIcon = "❄️"
	case "windy":
		conditionIcon = "💨"
	default:
		conditionIcon = "🌤️"
	}
	fmt.Printf("  %s 天气: %s\n", conditionIcon, response.Output.Condition)

	if response.Output.Description != "" {
		fmt.Printf("  📝 描述: %s\n", response.Output.Description)
	}

	if response.Output.HasAlert {
		fmt.Printf("  ⚠️  注意: 有天气预警\n")
	} else {
		fmt.Printf("  ✅ 无天气预警\n")
	}

	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("原始 JSON Answer:\n%s\n", response.ReActResponse.Answer)

	if err := multiLog.Close(); err != nil {
		fmt.Printf("Error closing logger: %v\n", err)
	}
}
