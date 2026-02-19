package main

import (
	"context"
	"fmt"
	// "os"

	"github.com/dreamzero-oxm/go-react-agent/agent"
	"github.com/dreamzero-oxm/go-react-agent/llm"
	"github.com/dreamzero-oxm/go-react-agent/logger"
	"github.com/dreamzero-oxm/go-react-agent/tools"
)

// Activity 表示一个活动安排
type Activity struct {
	Time     string `json:"time" agent:"desc:活动时间;required:true"`
	Location string `json:"location" agent:"desc:活动地点;required:true"`
	Action   string `json:"action" agent:"desc:活动内容;required:true"`
}

// DailyPlan 表示一天的行程计划
// 演示嵌套结构体和 slice 类型的使用
type DailyPlan struct {
	Date        string     `json:"date" agent:"desc:日期(YYYY-MM-DD);required:true"`
	City        string     `json:"city" agent:"desc:所在城市;required:true"`
	Weather     string     `json:"weather" agent:"desc:预计天气"`
	Activities  []Activity `json:"activities" agent:"desc:当天的活动列表"`
}

// TravelPlan 表示完整的旅游计划
// 演示嵌套结构体和多层嵌套的使用
type TravelPlan struct {
	Destination string      `json:"destination" agent:"desc:目的地;required:true"`
	Duration    int         `json:"duration" agent:"desc:行程天数;required:true;range:1,30"`
	Budget      float64     `json:"budget" agent:"desc:预估预算(元)"`
	Summary     string      `json:"summary" agent:"desc:行程摘要;required:true"`
	DailyPlans  []DailyPlan `json:"daily_plans" agent:"desc:每日行程安排"`
	Tips        []string    `json:"tips" agent:"desc:旅行小贴士"`
}

func main() {
	// Setup logging
	multiLog := logger.NewMultiLogger()
	multiLog.SetLevel(logger.LevelInfo)
	multiLog.AddConsoleLogger(true)

	// Get API key from environment
	// apiKey := os.Getenv("OPENAI_API_KEY")
	// if apiKey == "" {
	// 	fmt.Println("Please set OPENAI_API_KEY environment variable")
	// 	os.Exit(1)
	// }

	// // Configure LLM
	// llmConfig := &llm.LLMConfig{
	// 	APIKey:      apiKey,
	// 	BaseURL:     "https://api.openai.com/v1/chat/completions",
	// 	Model:       "gpt-4",
	// 	Temperature: 0.7,
	// 	MaxTokens:   3000,
	// }
	// // Configure LLM
	llmConfig := &llm.LLMConfig{
		APIKey:      "API_KEY",
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

	// Run with planning and structured output
	ctx := context.Background()
	query := "帮我规划一个从明天开始的3天2夜的潮州旅游行程，包括海边活动"

	fmt.Printf("╔══════════════════════════════════════════╗\n")
	fmt.Printf("║   Plan Agent 结构化输出示例             ║\n")
	fmt.Printf("╚══════════════════════════════════════════╝\n\n")
	fmt.Printf("Query: %s\n\n", query)

	// 使用泛型方法获取结构化输出
	response, plan, err := agent.RunStructuredWithPlan[TravelPlan](planningAgent, ctx, query)
	if err != nil {
		panic(err)
	}

	// 打印执行计划
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Execution Plan:\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	printPlan(plan)

	// 打印结构化输出
	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Structured Travel Plan:\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("🎯 目的地: %s\n", response.Output.Destination)
	fmt.Printf("📅 行程天数: %d 天\n", response.Output.Duration)
	if response.Output.Budget > 0 {
		fmt.Printf("💰 预估预算: ¥%.0f\n", response.Output.Budget)
	}
	fmt.Printf("\n📋 行程摘要:\n%s\n\n", response.Output.Summary)

	for i, daily := range response.Output.DailyPlans {
		fmt.Printf("━━━ 第 %d 天 (%s) ━━━\n", i+1, daily.Date)
		fmt.Printf("📍 城市: %s\n", daily.City)
		if daily.Weather != "" {
			fmt.Printf("🌤️  预计天气: %s\n", daily.Weather)
		}
		fmt.Printf("📝 活动安排:\n")
		for j, activity := range daily.Activities {
			fmt.Printf("   %d. %s - %s @ %s\n", j+1, activity.Time, activity.Action, activity.Location)
		}
		fmt.Println()
	}

	if len(response.Output.Tips) > 0 {
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("💡 旅行小贴士:\n")
		for _, tip := range response.Output.Tips {
			fmt.Printf("   • %s\n", tip)
		}
	}

	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("原始 JSON Answer:\n%s\n", response.ReActResponse.Answer)
}

func printPlan(plan *agent.Plan) {
	if plan == nil {
		fmt.Println("No plan available")
		return
	}
	fmt.Printf("📊 状态: %s\n", plan.Status)
	fmt.Printf("🧠 推理: %s\n\n", plan.Reasoning)
	for i, step := range plan.Steps {
		statusIcon := "⏳"
		switch step.Status {
		case "completed":
			statusIcon = "✅"
		case "failed":
			statusIcon = "❌"
		case "in_progress":
			statusIcon = "🔄"
		}
		fmt.Printf("%d. [%s] %s\n", i+1, statusIcon+" "+step.Status, step.Description)
		if step.Tool != "" {
			fmt.Printf("   🔧 工具: %s\n", step.Tool)
		}
		if step.Result != "" {
			fmt.Printf("   📤 结果: %s\n", step.Result)
		}
	}
}
