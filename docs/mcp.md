# MCP (Model Context Protocol) 集成文档

## 概述

本项目实现了完整的 MCP (Model Context Protocol) 支持，使 Agent 能够自动发现和使用来自 MCP Servers 的工具、资源和提示模板。

## 功能特性

### 协议层实现

- **JSON-RPC 2.0 通信**：完整的请求/响应/通知序列化支持
- **生命周期管理**：初始化握手（initialize → initialized）、能力协商
- **消息路由**：区分请求、响应、通知三类消息的处理

### 传输层支持

| 方式   | 适用场景       | 关键点                     |
| ------ | -------------- | -------------------------- |
| Stdio  | 本地工具       | 启动子进程，管理 stdin/stdout |
| SSE    | 远程服务       | HTTP 长连接，处理 Server-Sent Events |

### 核心能力对接

- **Tools（工具调用）**
  - 发现：tools/list 获取可用工具列表
  - 调用：tools/call 执行工具并获取结果
  - Schema 解析：处理工具的参数定义（JSON Schema）

- **Resources（资源访问）**
  - 订阅：resources/subscribe 监听资源变化
  - 读取：resources/read 获取资源内容
  - 模板解析：处理 URI 模板（如 file://{path}）

- **Prompts（提示模板）**
  - 获取：prompts/get 加载预定义提示
  - 参数填充：处理模板变量替换

## 架构设计

```
┌─────────────────┐
│   ReAct Agent   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  MCP Manager    │
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
┌────────┐  ┌────────┐
│ Client │  │ Client │
└───┬────┘  └───┬────┘
    │           │
    ▼           ▼
┌────────┐  ┌────────┐
│Stdio   │  │ SSE    │
│Transport│  │Transport│
└────────┘  └────────┘
```

## 快速开始

### 1. 配置 MCP Server

#### 使用命令行工具

```bash
# 添加 stdio 传输的 MCP Server
./mcp-cmd add --name filesystem \
  --command "npx" \
  --args "-y,@modelcontextprotocol/server-filesystem,/path/to/workspace"

# 添加 SSE 传输的 MCP Server
./mcp-cmd add --name custom-api \
  --transport "sse" \
  --url "http://localhost:3000/sse" \
  --headers "Authorization:Bearer ${API_KEY}"

# 列出所有配置的服务器
./mcp-cmd list

# 查看服务器状态
./mcp-cmd status

# 禁用服务器
./mcp-cmd disable filesystem

# 启用服务器
./mcp-cmd enable filesystem

# 移除服务器
./mcp-cmd remove filesystem
```

#### 使用配置文件

创建 `~/.go-react-agent/mcp.json` 或项目级 `.go-react-agent/mcp.json`：

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/home/user/workspace"],
      "env": {
        "NODE_ENV": "production"
      },
      "disabled": false,
      "timeout": 30000
    },
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "${GITHUB_TOKEN}"
      }
    },
    "custom-server": {
      "transport": "sse",
      "url": "http://localhost:3000/sse",
      "headers": {
        "Authorization": "Bearer ${API_KEY}"
      }
    }
  }
}
```

配置优先级：项目级配置 > 全局配置

### 2. 在 Agent 中启用 MCP

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
        APIKey:      os.Getenv("OPENAI_API_KEY"),
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

    // 方法 1: 使用 NewAgentWithMCP
    config := agent.DefaultConfig()
    config.MCPConfig.Enabled = true

    mcpAgent, err := agent.NewAgentWithMCP(openaiLLM, config, log)
    if err != nil {
        panic(err)
    }
    defer mcpAgent.Close()

    // 方法 2: 手动启用 MCP
    // reactAgent := agent.NewReActAgent(openaiLLM, config, log)
    // if err := reactAgent.WithMCPIntegration(); err != nil {
    //     panic(err)
    // }

    ctx := context.Background()
    response, err := mcpAgent.Run(ctx, "Read the README.md file")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Answer: %s\n", response.Answer)
}
```

### 3. 手动管理 MCP 工具

```go
// 手动管理 MCP Manager
config, _ := mcp.LoadConfig()
mcpManager := mcp.NewManager(config)

if err := mcpManager.Start(); err != nil {
    panic(err)
}
defer mcpManager.Stop()

// 获取状态
statuses := mcpManager.GetStatus()
for _, status := range statuses {
    fmt.Printf("Server: %s, Status: %s\n", status.Name, status.Status)
    fmt.Printf("  Tools: %d\n", len(status.Tools))
}

// 手动注册工具
agent := agent.NewReActAgent(llm, config, log)
if err := agent.WithMCPManager(mcpManager); err != nil {
    panic(err)
}
```

## API 参考

### Agent 集成 API

#### `NewAgentWithMCP(llm, config, log)`

创建一个自动启用 MCP 集成的 Agent。

**参数：**
- `llm`: LLM 接口实例
- `config`: Agent 配置（MCPConfig.Enabled 将自动设置为 true）
- `log`: Logger 实例

**返回：**
- `*agent.ReActAgent`: 启用了 MCP 的 Agent 实例
- `error`: 错误信息

#### `(*ReActAgent).WithMCPIntegration()`

为现有 Agent 启用 MCP 集成。

**返回：**
- `error`: 错误信息

#### `(*ReActAgent).WithMCPManager(manager)`

使用自定义 MCP Manager 注册工具。

**参数：**
- `manager`: MCP Manager 实例

**返回：**
- `error`: 错误信息

#### `GetMCPStatus()`

获取所有 MCP 服务器的状态。

**返回：**
- `[]mcp.ServerStatus`: 服务器状态列表
- `error`: 错误信息

### 配置 API

#### `mcp.LoadConfig()`

加载 MCP 配置文件（项目级和全局）。

**返回：**
- `*mcp.Config`: 配置对象
- `error`: 错误信息

#### `(*Config).Save()`

保存配置到文件。

**返回：**
- `error`: 错误信息

### Manager API

#### `mcp.NewManager(config)`

创建新的 MCP Manager。

**参数：**
- `config`: MCP 配置

**返回：**
- `*mcp.Manager`: Manager 实例

#### `(*Manager).Start()`

启动所有启用的 MCP 服务器。

**返回：**
- `error`: 错误信息

#### `(*Manager).Stop()`

停止所有 MCP 服务器。

**返回：**
- `error`: 错误信息

#### `(*Manager).GetStatus()`

获取所有服务器的状态。

**返回：**
- `[]ServerStatus`: 服务器状态列表

#### `(*Manager).GetRegistry()`

获取 MCP 工具注册表。

**返回：**
- `*MCPToolRegistry`: 工具注册表

## ServerStatus 结构

```go
type ServerStatus struct {
    Name        string         // 服务器名称
    Type        string         // 传输类型 (stdio/sse)
    Status      string         // 状态 (running/disabled/failed/initializing)
    Error       string         // 错误信息
    Tools       []ToolInfo     // 可用工具
    Resources   []ResourceInfo // 可用资源
    Prompts     []PromptInfo   // 可用提示
    Command     string         // 命令 (stdio)
    URL         string         // URL (sse)
}
```

## 配置说明

### 配置文件结构

```go
type Config struct {
    MCPServers map[string]ServerConfig `json:"mcpServers"`
}

type ServerConfig struct {
    Command   string            `json:"command,omitempty"`
    Args      []string          `json:"args,omitempty"`
    Env       map[string]string `json:"env,omitempty"`
    Transport string            `json:"transport,omitempty"`
    URL       string            `json:"url,omitempty"`
    Headers   map[string]string `json:"headers,omitempty"`
    Disabled  bool              `json:"disabled,omitempty"`
    Timeout   int               `json:"timeout,omitempty"`
}
```

### 环境变量扩展

配置支持环境变量替换：

```json
{
  "env": {
    "GITHUB_TOKEN": "${GITHUB_TOKEN}"
  }
}
```

运行时会自动替换为实际的环境变量值。

## 错误处理

MCP 集成包含完善的错误处理机制：

1. **配置错误**：服务器配置无效或缺失
2. **连接错误**：无法连接到 MCP Server
3. **初始化错误**：握手失败
4. **工具注册错误**：工具参数解析失败

所有错误都会通过 Logger 输出，不会中断 Agent 的正常运行。

## 最佳实践

### 1. 配置管理

- 使用项目级配置来存储项目特定的 MCP Servers
- 使用全局配置来存储通用的 MCP Servers
- 定期检查服务器状态，禁用不常用的服务器

### 2. 性能优化

- 设置合理的超时时间（默认 30 秒）
- 禁用不需要的服务器以减少启动时间
- 对于本地工具，优先使用 stdio 传输

### 3. 安全性

- 不要在配置文件中硬编码敏感信息
- 使用环境变量存储 API 密钥
- 限制 MCP Server 的访问权限

## 故障排查

### 服务器启动失败

```bash
# 检查服务器状态
./mcp-cmd status

# 查看详细日志
# 检查命令是否正确
# 验证环境变量是否设置
```

### 工具未注册

```go
// 检查 Agent 是否启用了 MCP
if !agent.IsMCPEnabled() {
    fmt.Println("MCP is not enabled")
}

// 查看 Agent 日志，寻找 "Failed to register MCP tool" 消息
```

### 连接超时

```go
// 增加超时时间
serverConfig := mcp.ServerConfig{
    Timeout: 60000, // 60 秒
}
```

## 示例

完整示例请参考 [example/mcp_example.go](../example/mcp_example.go)

## 兼容性

- MCP 协议版本：2024-11-05
- 支持的 MCP Servers：所有符合 MCP 规范的 servers
- Go 版本：1.21+

## 贡献

欢迎贡献代码和报告问题！

## 许可证

MIT License
