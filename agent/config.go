package agent

import (
	"reflect"
	"time"
)

// OutputConfig defines the configuration for structured output behavior.
type OutputConfig struct {
	// OutputType is the target struct type for structured output (using reflect.Type)
	OutputType reflect.Type `json:"-"`

	// OutputSchema is the generated JSON schema string
	OutputSchema string `json:"output_schema,omitempty"`

	// EnableStructuredOutput enables structured output mode
	EnableStructuredOutput bool `json:"enable_structured_output"`

	// MaxNestingDepth is the maximum nesting depth for schema generation
	// (prevents overly long prompts)
	MaxNestingDepth int `json:"max_nesting_depth"`

	// MaxParseRetries is the maximum number of retries for parsing structured output
	MaxParseRetries int `json:"max_parse_retries"`
}

// Config holds the configuration for the ReAct agent.
type Config struct {
	// MaxIterations is the maximum number of reasoning-acting cycles
	MaxIterations int `json:"max_iterations"`
	// Timeout is the maximum duration for agent execution
	Timeout time.Duration `json:"timeout"`
	// Temperature controls the randomness of LLM responses (0.0 to 1.0)
	Temperature float64 `json:"temperature"`
	// MaxTokens is the maximum number of tokens in LLM responses
	MaxTokens int `json:"max_tokens"`
	// Parser is the response parser for LLM output
	Parser ResponseParser `json:"-"`
	// PlanConfig contains planning feature configuration
	PlanConfig *PlanConfig `json:"plan_config,omitempty"`
	// Output contains structured output configuration
	Output *OutputConfig `json:"output,omitempty"`
	// MCPConfig contains MCP integration configuration
	MCPConfig *MCPConfig `json:"mcp_config,omitempty"`
	// SkillConfig contains skill integration configuration
	SkillConfig *SkillConfig `json:"skill_config,omitempty"`
	// Debug enables verbose logging of each step (thought, action, observation)
	Debug bool `json:"debug,omitempty"`
}

// MCPConfig contains MCP (Model Context Protocol) integration configuration.
type MCPConfig struct {
	// Enabled enables MCP integration
	Enabled bool `json:"enabled"`
	// AutoLoadConfig automatically loads MCP config from files
	AutoLoadConfig bool `json:"auto_load_config"`
	// GlobalConfigPath is the path to global MCP config file (default: ~/.go-react-agent/mcp.json)
	GlobalConfigPath string `json:"global_config_path,omitempty"`
	// ProjectConfigPath is the path to project MCP config file (default: .go-react-agent/mcp.json)
	ProjectConfigPath string `json:"project_config_path,omitempty"`
}

// SkillConfig contains Claude Code Skills integration configuration.
type SkillConfig struct {
	// Enabled enables skill integration
	Enabled bool `json:"enabled"`
	// AutoLoadSkills automatically loads skills from directories
	AutoLoadSkills bool `json:"auto_load_skills"`
	// MaxSkillsPerQuery is the maximum number of skills to inject per query
	MaxSkillsPerQuery int `json:"max_skills_per_query"`
	// GlobalSkillsDir is the global skills directory (default: ~/.claude/skills/)
	GlobalSkillsDir string `json:"global_skills_dir"`
	// ProjectSkillsDir is the project skills directory (default: .claude/skills/)
	ProjectSkillsDir string `json:"project_skills_dir"`
}

// DefaultConfig returns a Config with sensible default values.
//
// Defaults:
//   - MaxIterations: 10
//   - Timeout: 10 minutes
//   - Temperature: 0.7
//   - MaxTokens: 4096
//   - Parser: JSONParser
//   - EnableStructuredOutput: false
//   - MaxNestingDepth: 5
//   - MaxParseRetries: 3
//   - MCPEnabled: false
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
		MCPConfig: &MCPConfig{
			Enabled:           false,
			AutoLoadConfig:    true,
			GlobalConfigPath:  "~/.go-react-agent/mcp/mcp.json",
			ProjectConfigPath: ".go-react-agent/mcp/mcp.json",
		},
		SkillConfig: &SkillConfig{
			Enabled:           false,
			AutoLoadSkills:    true,
			MaxSkillsPerQuery: 3,
			GlobalSkillsDir:   "~/.go-react-agent/skills/",
			ProjectSkillsDir:  ".go-react-agent/skills/",
		},
	}
}
