package agent

import (
	"fmt"

	"github.com/dreamzero-oxm/go-react-agent/llm"
	"github.com/dreamzero-oxm/go-react-agent/logger"
	"github.com/dreamzero-oxm/go-react-agent/skills"
)

// WithSkillIntegration adds Claude Code Skills integration to the agent.
//
// This function enables the agent to load skills from SKILL.md files and
// inject their content as context for the LLM. Skills provide domain
// knowledge and guidance rather than executable tools.
//
// Returns:
//   - error: An error if enabling skills fails.
//
// Usage:
//
//	agent := agent.NewReActAgent(llm, config, log)
//	if err := agent.WithSkillIntegration(); err != nil {
//	    log.Fatalf("Failed to enable skills: %v", err)
//	}
func (a *ReActAgent) WithSkillIntegration() error {
	if a.config.SkillConfig == nil || !a.config.SkillConfig.Enabled {
		return fmt.Errorf("skill integration is not enabled in config")
	}

	globalDir := a.config.SkillConfig.GlobalSkillsDir
	if globalDir == "" {
		globalDir = "~/.go-react-agent/skills/"
	}

	projectDir := a.config.SkillConfig.ProjectSkillsDir
	if projectDir == "" {
		projectDir = ".go-react-agent/skills/"
	}

	loadedSkills, err := skills.LoadSkills(globalDir, projectDir)
	if err != nil {
		return fmt.Errorf("failed to load skills: %w", err)
	}

	a.skills = loadedSkills

	a.logger.Info("Claude Code Skills loaded", map[string]interface{}{
		"count": len(loadedSkills),
	})

	// Log loaded skill names
	for _, skill := range loadedSkills {
		a.logger.Info("Loaded skill", map[string]interface{}{
			"name": skill.Name,
			"tags": skill.Tags,
		})
	}

	return nil
}

// NewAgentWithSkills creates a new agent with Claude Code Skills integration automatically enabled.
//
// This is a convenience function that creates an agent and enables skill integration
// in one step. Skills are automatically loaded from ~/.claude/skills/ and .claude/skills/.
//
// Parameters:
//   - llm: The LLM instance.
//   - config: The agent configuration.
//   - log: The logger instance.
//
// Returns:
//   - *ReActAgent: The created agent.
//   - error: An error if creation or skill initialization fails.
//
// Usage:
//
//	config := agent.DefaultConfig()
//	config.SkillConfig.Enabled = true
//
//	agent, err := agent.NewAgentWithSkills(llm, config, log)
//	if err != nil {
//	    log.Fatalf("Failed to create agent: %v", err)
//	}
func NewAgentWithSkills(llm llm.LLM, config *Config, log logger.Logger) (*ReActAgent, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if config.SkillConfig == nil {
		config.SkillConfig = &SkillConfig{
			Enabled:           true,
			AutoLoadSkills:    true,
			MaxSkillsPerQuery: 3,
		}
	} else {
		config.SkillConfig.Enabled = true
		if config.SkillConfig.MaxSkillsPerQuery == 0 {
			config.SkillConfig.MaxSkillsPerQuery = 3
		}
	}

	agent := NewReActAgent(llm, config, log)

	if err := agent.WithSkillIntegration(); err != nil {
		return nil, err
	}

	return agent, nil
}

// SetSkills sets the skills to use for this agent.
//
// This allows manual control over which skills are available, bypassing
// automatic loading from directories.
//
// Parameters:
//   - skillsList: List of skills to use.
func (a *ReActAgent) SetSkills(skillsList []*skills.Skill) {
	a.skills = make(map[string]*skills.Skill)
	for _, skill := range skillsList {
		a.skills[skill.Name] = skill
	}
}

// GetSkills returns the currently loaded skills.
//
// Returns:
//   - []*skills.Skill: List of loaded skills.
func (a *ReActAgent) GetSkills() []*skills.Skill {
	result := make([]*skills.Skill, 0, len(a.skills))
	for _, skill := range a.skills {
		result = append(result, skill)
	}
	return result
}

// IsSkillEnabled returns whether skill integration is enabled for this agent.
//
// Returns:
//   - bool: True if skills are enabled.
func (a *ReActAgent) IsSkillEnabled() bool {
	return a.config.SkillConfig != nil && a.config.SkillConfig.Enabled
}
