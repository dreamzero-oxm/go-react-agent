# Claude Code Skills 集成文档

## 概述

go-react-agent 支持官方 Claude Code Skills 格式，允许你通过 SKILL.md 文件为 Agent 提供领域知识和指导。Skills 使用 Markdown 文件定义，可以被 Agent 引用以提供更准确、更专业的回答。

### 什么是 Claude Code Skills？

Claude Code Skills 是一种通过 Markdown 文件为 Claude 提供上下文和指导的标准格式。每个 Skill 包含：
- **YAML 元数据**：名称、版本、描述、标签
- **Markdown 内容**：详细的指导、示例、最佳实践

与可执行工具不同，Skills 提供的是**知识和指导**，而非执行操作。

---

## 功能特性

### 双目录支持
- **全局**: `~/.claude/skills/` - 应用于所有项目
- **项目**: `.claude/skills/` - 项目特定，覆盖全局同名技能

### 自动上下文注入
- Skills 内容自动注入到 Agent 的系统提示中
- LLM 可以引用 Skills 中的知识和指导
- 支持限制每次查询注入的技能数量

### 智能 Skill 选择
- 基于查询内容自动选择相关 Skills
- 使用标签、描述和名称进行匹配
- 可配置每次查询的最大技能数

---

## SKILL.md 文件格式

### 基本结构

```
my-custom-skill/
├── SKILL.md          # 必需 - 技能定义
├── helpers.py        # 可选 - 支持脚本
└── resources/        # 可选 - 模板、示例数据
```

### SKILL.md 格式规范

```yaml
---
name: my-custom-skill           # 必需：技能名称（必须与文件夹名匹配）
version: 1.0                    # 可选：版本号
description: |                  # 必需：技能描述（Claude 用于决定何时激活）
  A brief description of what this skill does.
  Multiple lines are supported.
tags:                           # 可选：分类标签
  - documentation
  - markdown
---

# Markdown 内容区域

在这里提供详细的指导、示例和最佳实践。
```

### 完整示例

```yaml
---
name: go-expert
version: 1.0
description: |
  Provides expert knowledge about Go programming language including
  best practices, idioms, concurrency patterns, and common pitfalls.
tags:
  - go
  - golang
  - programming
---

# Go Expert Skill

This skill provides expert knowledge about the Go programming language.

## Key Concepts

### Concurrency Patterns

#### Goroutines
```go
go func() {
    // Do work concurrently
}()
```

### Common Pitfalls

1. Not checking errors
2. Goroutine leaks
3. Channel misuse

## Resources

- [Effective Go](https://go.dev/doc/effective_go)
```

---

## 快速开始

### 1. 创建 Skill 文件

创建 `~/.claude/skills/my-skill/SKILL.md`:

```yaml
---
name: my-skill
version: 1.0
description: My custom skill for specific domain knowledge
tags:
  - custom
  - domain
---

# My Custom Skill

Detailed instructions here...
```

### 2. 在 Agent 中启用

```go
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

    llmConfig := &llm.LLMConfig{
        APIKey:  os.Getenv("OPENAI_API_KEY"),
        BaseURL: "https://api.openai.com/v1/chat/completions",
        Model:   "gpt-3.5-turbo",
    }

    openaiLLM, _ := llm.NewOpenAILLM(llmConfig)
    defer openaiLLM.Close()

    // 创建 Agent 并启用 Claude Code Skills
    config := agent.DefaultConfig()
    config.SkillConfig.Enabled = true

    skillAgent, err := agent.NewAgentWithSkills(openaiLLM, config, log)
    if err != nil {
        panic(err)
    }
    defer skillAgent.Close()

    // 查询时 Agent 会自动引用相关 Skills
    ctx := context.Background()
    response, err := skillAgent.Run(ctx, "How do I properly handle errors in Go?")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Answer: %s\n", response.Answer)
}
```

---

## 工作原理

go-react-agent 使用 RAG (Retrieval-Augmented Generation) 模式实现 Claude Code Skills：

### 1. 系统提示注入（仅元数据）

当启用 Skills 时，系统提示中只包含 Skills 的**元数据**：

```
## Available Skills

- **go-expert** (v1.0)
  - Description: Provides expert knowledge about Go programming language
  - Tags: go, golang, programming

- **api-design** (v1.0)
  - Description: REST API design guidance
  - Tags: api, rest, design

To use a skill, respond with:
{"action": {"name": "use_skill", "input": {"skill_name": "skill-name"}}}
```

### 2. Agent 显式请求使用 Skill

当 Agent 需要使用某个 Skill 时，它会返回特殊的 action：

```json
{
  "thoughts": [{"content": "This question requires Go expertise"}],
  "action": {
    "name": "use_skill",
    "input": {"skill_name": "go-expert"}
  }
}
```

### 3. 系统加载完整 Skill 内容

检测到 `use_skill` action 后，系统会：
1. 加载该 Skill 的完整 SKILL.md 内容
2. 创建新的提示：Skill 内容 + 原始查询
3. 调用 LLM 生成基于 Skill 的回答

### 4. Agent 提供最终答案

Agent 基于完整的 Skill 内容提供准确的回答。

### 示例流程

```
用户: "Go 中如何正确处理错误？"

Agent (第1轮):
{
  "thoughts": ["这个问题需要 Go 专家知识"],
  "action": {"name": "use_skill", "input": {"skill_name": "go-expert"}}
}

系统:
1. 检测到 use_skill
2. 加载完整的 go-expert/SKILL.md
3. 创建新提示并获取 LLM 回答

Agent (第2轮):
{
  "answer": "根据 go-expert 技能，Go 中的错误处理遵循以下模式..."
}
```

---

## API 参考

### Agent 集成 API

#### `NewAgentWithSkills(llm, config, log)`

创建一个自动启用 Claude Code Skills 的 Agent。

**参数：**
- `llm`: LLM 接口实例
- `config`: Agent 配置（SkillConfig.Enabled 将自动设置为 true）
- `log`: Logger 实例

**返回：**
- `*agent.ReActAgent`: 启用了 Skills 的 Agent 实例
- `error`: 错误信息

#### `(*ReActAgent).WithSkillIntegration()`

为现有 Agent 启用 Skill 集成。

**返回：**
- `error`: 错误信息

#### `(*ReActAgent).GetSkills()`

获取当前加载的所有 Skills。

**返回：**
- `[]*skills.Skill`: 加载的技能列表

#### `(*ReActAgent).SelectSkillsForQuery(query)`

根据查询选择相关的 Skills。

**参数：**
- `query`: 用户查询字符串

**返回：**
- `[]*skills.Skill`: 相关技能列表

#### `(*ReActAgent).IsSkillEnabled()`

返回 Skill 集成是否启用。

**返回：**
- `bool`: 如果启用则返回 true

### Skills 加载 API

#### `skills.LoadSkills()`

从全局和项目目录加载所有 Skills。

**返回：**
- `[]*skills.Skill`: 加载的技能列表
- `error`: 错误信息

#### `skills.LoadSkill(dirPath)`

从指定目录加载单个 Skill。

**参数：**
- `dirPath`: Skill 目录路径

**返回：**
- `*skills.Skill`: 加载的技能
- `error`: 错误信息

### Skill 选择 API

#### `skills.NewSelector(skills).SelectSkills(query, maxSkills)`

根据查询选择相关 Skills。

**参数：**
- `query`: 查询字符串
- `maxSkills`: 最大返回技能数（0 = 无限制）

**返回：**
- `[]*skills.Skill`: 选中的技能列表

---

## 配置说明

### Agent 配置

```go
type SkillConfig struct {
    // Enabled enables skill integration
    Enabled bool `json:"enabled"`
    // AutoLoadSkills automatically loads skills from directories
    AutoLoadSkills bool `json:"auto_load_skills"`
    // MaxSkillsPerQuery is the maximum number of skills to inject per query
    MaxSkillsPerQuery int `json:"max_skills_per_query"`
}
```

### 默认配置

```go
config.SkillConfig = &SkillConfig{
    Enabled:          false,
    AutoLoadSkills:   true,
    MaxSkillsPerQuery: 3,  // 默认每次最多注入 3 个技能
}
```

---

## 目录结构

```
~/.claude/skills/                    # 全局 Claude Code Skills
  ├── go-expert/
  │   └── SKILL.md
  ├── api-design/
  │   └── SKILL.md
  └── testing/
      ├── SKILL.md
      └── resources/
          └── test-templates.md

.claude/skills/                      # 项目特定 Skills
  └── project-guidelines/
      └── SKILL.md
```

---

## Skill 选择机制

Skills 基于以下因素自动选择：

1. **标签匹配** - 完全匹配：+10 分，部分匹配：+5 分
2. **名称匹配** - 完全匹配：+8 分，部分匹配：+3 分
3. **描述关键词** - 每个匹配关键词：+1 分

选择分数最高的 Skills，最多返回 `MaxSkillsPerQuery` 个。

---

## 最佳实践

### Skill 设计

1. **单一职责** - 每个 Skill 应专注于一个特定领域
2. **清晰描述** - `description` 字段至关重要，Claude 使用它来决定何时激活 Skill
3. **合理使用标签** - 标签应该能准确分类 Skill
4. **渐进式披露** - 在 SKILL.md 中提供概览，详细信息可放在 resources/ 目录

### 内容组织

1. **保持简洁** - SKILL.md 应该少于 500 行
2. **使用示例** - 包含可运行的代码示例
3. **结构化内容** - 使用标题、列表、代码块
4. **避免过时 API** - 使用最新的库和接口

### 性能优化

1. **限制 Skill 数量** - 设置合理的 `MaxSkillsPerQuery`
2. **精简内容** - 每个 Skill 的内容会注入到提示中
3. **项目覆盖** - 使用项目 Skills 覆盖不需要的全局 Skills

---

## 故障排查

### Skills 未加载

```bash
# 检查目录是否存在
ls -la ~/.claude/skills/
ls -la .claude/skills/

# 检查 SKILL.md 格式
cat ~/.claude/skills/my-skill/SKILL.md
```

### Skill 内容未生效

1. 确认 `config.SkillConfig.Enabled = true`
2. 检查 Skill 的 `description` 是否与查询相关
3. 查看 Agent 日志，确认 Skills 已加载
4. 尝试增加 `MaxSkillsPerQuery`

### YAML 解析错误

确保：
- YAML 块以 `---` 开始和结束
- `name` 字段与文件夹名称一致
- 缩进使用空格，不要使用 Tab

---

## 示例

完整示例请参考：
- [examples/claude-skills/go-expert/SKILL.md](../examples/claude-skills/go-expert/SKILL.md)
- [examples/claude-skills/api-design/SKILL.md](../examples/claude-skills/api-design/SKILL.md)
- [examples/claude-skills/example.go](../examples/claude-skills/example.go)

---

## 兼容性

- Go 版本：1.21+
- 支持的 Skill 格式：官方 Claude Code Skills (SKILL.md)
- YAML 解析：yaml.v3

---

## 贡献

欢迎贡献代码和报告问题！

---

## 许可证

MIT License
