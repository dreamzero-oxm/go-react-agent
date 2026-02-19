package agent

import (
	"reflect"
	"time"
)

// OutputConfig 定义结构化输出的配置
type OutputConfig struct {
	// OutputType 是目标结构体的类型 (使用 reflect.Type)
	OutputType reflect.Type `json:"-"`

	// OutputSchema 是生成的 JSON Schema 字符串
	OutputSchema string `json:"output_schema,omitempty"`

	// EnableStructuredOutput 启用结构化输出
	EnableStructuredOutput bool `json:"enable_structured_output"`

	// MaxNestingDepth 最大嵌套层级（防止 Prompt 过长）
	MaxNestingDepth int `json:"max_nesting_depth"`

	// MaxParseRetries 最大解析重试次数
	MaxParseRetries int `json:"max_parse_retries"`
}

type Config struct {
	MaxIterations int            `json:"max_iterations"`
	Timeout       time.Duration  `json:"timeout"`
	Temperature   float64        `json:"temperature"`
	MaxTokens     int            `json:"max_tokens"`
	Parser        ResponseParser `json:"-"` // Response parser for LLM output
	PlanConfig    *PlanConfig    `json:"plan_config,omitempty"` // Planning feature configuration
	Output        *OutputConfig  `json:"output,omitempty"`      // Structured output configuration
}

func DefaultConfig() *Config {
	return &Config{
		MaxIterations: 10,
		Timeout:       10 * time.Minute,
		Temperature:   0.7,
		MaxTokens:     4096,
		Parser:        NewJSONParser(),
		Output: &OutputConfig{
			EnableStructuredOutput: false,
			MaxNestingDepth:        5,
			MaxParseRetries:        3,
		},
	}
}
