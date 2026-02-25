// Package skills provides Claude Code Skills support for go-react-agent.
// Claude Code Skills use markdown files (SKILL.md) to provide context
// and guidance to the agent.
package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Skill represents a loaded Claude Code Skill.
type Skill struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Content     string   `json:"content"`
	FilePath    string   `json:"file_path"` // Path to the skill directory
}

// SkillMetadata represents the YAML frontmatter from SKILL.md.
type SkillMetadata struct {
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
}

// LoadSkill loads a single skill from a directory.
//
// The directory must contain a SKILL.md file with YAML frontmatter.
//
// Parameters:
//   - dirPath: Path to the skill directory.
//
// Returns:
//   - *Skill: The loaded skill.
//   - error: An error if loading fails.
func LoadSkill(dirPath string) (*Skill, error) {
	skillMDPath := filepath.Join(dirPath, "SKILL.md")
	if _, err := os.Stat(skillMDPath); err != nil {
		return nil, fmt.Errorf("SKILL.md not found in %s: %w", dirPath, err)
	}

	content, err := os.ReadFile(skillMDPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SKILL.md: %w", err)
	}

	// Parse YAML frontmatter
	metadata, contentBody, err := parseFrontmatter(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Extract skill name from directory name if not in metadata
	skillName := metadata.Name
	if skillName == "" {
		skillName = filepath.Base(dirPath)
	}

	return &Skill{
		Name:        skillName,
		Version:     metadata.Version,
		Description: metadata.Description,
		Tags:        metadata.Tags,
		Content:     strings.TrimSpace(contentBody),
		FilePath:    dirPath,
	}, nil
}

// LoadSkills loads all skills from global and project directories.
// Project skills override global skills with the same name.
//
// Parameters:
//   - globalDir: Path to global skills directory (supports ~ for home dir)
//   - projectDir: Path to project skills directory (relative or absolute)
//
// Returns:
//   - map[string]*Skill: Map of loaded skills (name -> skill).
//   - error: An error if loading fails.
func LoadSkills(globalDir string, projectDir string) (map[string]*Skill, error) {
	skillsMap := make(map[string]*Skill)

	// Expand ~ in global directory path
	expandedGlobalDir := expandPath(globalDir)

	// First, load global skills
	globalSkills, err := loadSkillsFromDir(expandedGlobalDir)
	if err == nil {
		for _, skill := range globalSkills {
			skillsMap[skill.Name] = skill
		}
	}

	// Then, load project skills (overrides global)
	projectSkills, err := loadSkillsFromDir(projectDir)
	if err == nil {
		for _, skill := range projectSkills {
			skillsMap[skill.Name] = skill
		}
	}

	return skillsMap, nil
}

// expandPath expands ~ to the user's home directory.
//
// Parameters:
//   - path: The path to expand.
//
// Returns:
//   - string: The expanded path.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(homeDir, path[2:])
		}
	}
	return path
}

// loadSkillsFromDir loads all skills from a specific directory.
//
// Parameters:
//   - dir: Directory path to search for skills.
//
// Returns:
//   - []*Skill: List of loaded skills.
//   - error: An error if directory cannot be read.
func loadSkillsFromDir(dir string) ([]*Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	skills := make([]*Skill, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(dir, entry.Name())
		skill, err := LoadSkill(skillDir)
		if err != nil {
			// Log warning but continue loading other skills
			fmt.Printf("Warning: failed to load skill from %s: %v\n", skillDir, err)
			continue
		}

		skills = append(skills, skill)
	}

	return skills, nil
}

// parseFrontmatter parses YAML frontmatter from markdown content.
//
// Expected format:
//
//	---
//	name: skill-name
//	version: 1.0
//	description: |
//	  Multi-line description
//	tags:
//	  - tag1
//	  - tag2
//	---
//
//	Content here...
//
// Parameters:
//   - content: The markdown content with frontmatter.
//
// Returns:
//   - *SkillMetadata: The parsed metadata.
//   - string: The content body (without frontmatter).
//   - error: An error if parsing fails.
func parseFrontmatter(content string) (*SkillMetadata, string, error) {
	lines := strings.Split(content, "\n")

	if len(lines) < 3 || !strings.HasPrefix(lines[0], "---") {
		// No frontmatter, return empty metadata
		return &SkillMetadata{}, content, nil
	}

	// Find end of frontmatter
	endIndex := -1
	for i := 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "---") {
			endIndex = i
			break
		}
	}

	if endIndex == -1 {
		// No closing delimiter, treat as no frontmatter
		return &SkillMetadata{}, content, nil
	}

	// Extract frontmatter YAML
	frontmatter := strings.Join(lines[1:endIndex], "\n")

	var metadata SkillMetadata
	if err := yaml.Unmarshal([]byte(frontmatter), &metadata); err != nil {
		return nil, "", fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	// Extract content body
	contentBody := strings.Join(lines[endIndex+1:], "\n")

	return &metadata, strings.TrimSpace(contentBody), nil
}
