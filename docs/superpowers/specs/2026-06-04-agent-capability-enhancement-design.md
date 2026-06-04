# Agent 能力增强设计：持久记忆 + 工具扩展

## 背景

当前内置 Agent（builtin runtime）的能力集中在 cago `app/coding` 的工具循环中，支持文件读写、代码执行等基础操作。用户希望增强两个方面：

1. **持久记忆** — Agent 能记住跨会话的用户偏好、项目上下文、对话摘要
2. **工具调用扩展** — Agent 可调用自定义工具（搜索、API、文件系统扩展等）

## 方案选择

| 方案 | 描述 | 优点 | 缺点 |
|------|------|------|------|
| A. 渐进增强 | 新增 agent_memory + agent_tool 两个独立域 | 遵循现有架构，可独立交付 | 工作量较大 |
| B. 轻量注入 | 在 agents 表扩展字段 | 改动最小 | 不符合高内聚低耦合，扩展性差 |
| C. MCP 优先 | 通过 MCP 同时解决两类需求 | 生态兼容 | 依赖外部进程，范围大 |

**选定方案 A**：遵循项目 domain 分层模式，两个子项目可独立交付。

---

## 子项目一：持久记忆层（agent_memory）

### 数据模型

新增 `agent_memories` 表：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int64 PK | 自增主键 |
| agent_id | int64 | 关联 agents.id |
| scope | text | "user" / "project" / "session" |
| session_id | int64 | scope=session 时关联 chat_sessions.id，否则 0 |
| category | text | "preference" / "summary" / "context" / "custom" |
| key | text | 可选，用于 preference 类型的键值对 |
| content | text | 记忆正文 |
| source | text | "auto" / "manual" |
| status | int | ACTIVE / INACTIVE |
| createtime | bigint | 创建时间 |
| updatetime | bigint | 更新时间 |

**Scope 分区**：
- `user` — 用户偏好，跨会话持久（如"用户喜欢用 TypeScript"）
- `project` — 项目上下文，同一项目所有用户共享（如"本项目使用 Go 1.26 + Wails v2"）
- `session` — 会话摘要，单个会话结束时的总结

**Category 分类**：
- `preference` — 用户偏好，key-value 结构
- `summary` — 对话摘要，session 结束时自动生成
- `context` — 项目/代码上下文
- `custom` — 用户手动添加的自定义记忆

**索引**：
- `idx_agent_memories_agent_scope` on (agent_id, scope)
- `idx_agent_memories_session` on (session_id) where scope='session'

### 分层结构

```
internal/model/entity/agent_memory_entity/agent_memory.go   — 充血实体 + Check()
internal/repository/agent_memory_repo/agent_memory.go       — 接口 + 实现
internal/service/agent_memory_svc/agent_memory.go           — 应用服务
internal/app/agent_memory.go                                — Wails 绑定
```

### 服务接口

```go
type AgentMemorySvc interface {
    // 手动管理
    Create(ctx context.Context, req *CreateMemoryRequest) (*CreateMemoryResponse, error)
    Update(ctx context.Context, req *UpdateMemoryRequest) (*UpdateMemoryResponse, error)
    Delete(ctx context.Context, req *DeleteMemoryRequest) (*DeleteMemoryResponse, error)
    List(ctx context.Context, req *ListMemoryRequest) (*ListMemoryResponse, error)

    // 运行时注入
    LoadMemories(ctx context.Context, agentID int64, scope string, sessionID int64) ([]*MemoryItem, error)

    // 自动提取
    ExtractFromSession(ctx context.Context, sessionID int64) error
}
```

### 与 builtin runtime 的集成

1. 在 `builtin/runtime.go` 的 `Run()` 方法中，构建 `coding.Option` 时注入记忆：
   - 调用 `agent_memory_svc.LoadMemories()` 获取该 agent 的 user + project scope 记忆
   - 将记忆格式化为 system prompt 片段追加到 `coding.AppendSystem()`
2. Session 结束时（chat_svc 检测到 turn 正常结束），触发 `ExtractFromSession()` 异步提取摘要

### 自动记忆提取流程

1. Session 结束时（chat_svc 检测到 turn 正常结束），调用 `ExtractFromSession(sessionID)`
2. 该方法获取该 session 的对话历史，构造一个精简的 LLM 调用，让 LLM 提取关键信息
3. 提取结果写入 `agent_memories` 表，scope=session，category=summary，source=auto
4. 同时检查是否提取到用户偏好，如有则 upsert scope=user，category=preference

### 前端

- Agent 设置/编辑页新增"记忆"标签页
- 列表展示：按 scope 分组，显示 category、content、source、时间
- 支持手动添加/编辑/删除记忆条目
- 记忆条目可启用/禁用（Status 切换）

### 数据库迁移

新增迁移 append 到 `migrationList()` 末尾，创建 `agent_memories` 表及索引。

---

## 子项目二：工具扩展层（agent_tool）

### 核心思路

cago `coding.New()` 的工具循环是内置的，当前没有暴露 `WithTools` 扩展点。工具扩展层采用**运行时拦截 + 路由**的方式：

- 在 builtin runtime 的 `Run()` 中，将自定义工具定义注入 system prompt（让 LLM 知道可以调用这些工具）
- 在 `translate()` 中，当检测到自定义工具的 `ToolCall` 时，路由到对应的 `ToolExecutor` 执行
- 执行结果作为 `ToolResult` 事件返回，继续工具循环

### 数据模型

新增 `agent_tools` 表：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | int64 PK | 自增主键 |
| agent_id | int64 | 关联 agents.id（0 = 全局工具） |
| name | text | 工具名称，如 "web_search" |
| description | text | 工具描述，LLM 用于判断何时调用 |
| param_schema | text | JSON Schema，定义工具参数 |
| executor_type | text | "builtin" / "http" / "mcp" |
| executor_conf | text | JSON，执行器配置 |
| enabled | bool | 是否启用 |
| sort_order | int | 排序 |
| status | int | ACTIVE / INACTIVE |
| createtime | bigint | 创建时间 |
| updatetime | bigint | 更新时间 |

**ExecutorType 分类**：
- `builtin` — 进程内执行，代码实现（如 web_search、code_search）
- `http` — 调用外部 HTTP API
- `mcp` — 调用 MCP 工具服务器（预留，未来扩展）

**索引**：
- `idx_agent_tools_agent` on (agent_id)
- `uniq_agent_tools_name` on (agent_id, name) — 同一 agent 下工具名唯一

### 分层结构

```
internal/model/entity/agent_tool_entity/agent_tool.go
internal/repository/agent_tool_repo/agent_tool.go
internal/service/agent_tool_svc/agent_tool.go
internal/pkg/agentruntime/toolext/
  executor.go    — ToolExecutor 接口
  registry.go    — 工具注册表
  builtin.go     — 内置执行器（web_search 等）
  http.go        — HTTP 执行器
  prompt.go      — system prompt 生成
internal/app/agent_tool.go                      — Wails 绑定
```

### ToolExecutor 接口

```go
package toolext

// ToolExecutor 工具执行器接口。
type ToolExecutor interface {
    Execute(ctx context.Context, params json.RawMessage) (string, error)
}

// ToolDef 工具定义，用于注入 system prompt。
type ToolDef struct {
    Name        string
    Description string
    ParamSchema json.RawMessage
}
```

### 与 builtin runtime 的集成

1. **注入工具定义**：在 `builtin/runtime.go` 的 `Run()` 中，调用 `toolext.LoadTools(agentID)` 获取该 agent 启用的工具列表，将工具定义格式化为 function calling schema 追加到 system prompt。

2. **拦截工具调用**：在 `translate()` 函数中，当 `agent.EventPreToolUse` 的工具名匹配自定义工具时：
   - 不走默认的 cago 工具执行路径
   - 调用 `toolext.Execute(name, input)` 执行
   - 将结果作为 `ToolResult` 事件 emit

3. **实现方式**：在 cago agent 的事件循环中，自定义工具的调用需要通过 cago 的 `Steer` 机制注入结果——当检测到自定义工具的 `PreToolUse` 事件时，通过 `runner.Steer()` 将执行结果注入回对话流。如果 cago 不支持这种拦截模式，则采用**双循环**方案：外层循环管理自定义工具的执行，内层循环是 cago 的原生工具循环。

### 开箱即用的内置工具

| 工具名 | 描述 | 参数 |
|--------|------|------|
| `web_search` | 搜索互联网信息 | `query: string, max_results?: int` |
| `code_search` | 在项目代码库中搜索 | `query: string, file_pattern?: string` |
| `read_file` | 读取项目文件内容 | `path: string, start_line?: int, end_line?: int` |
| `list_files` | 列出目录结构 | `path: string, pattern?: string` |

### 前端

- Agent 设置/编辑页新增"工具"标签页
- 展示可用工具列表（全局 + 该 Agent 专属）
- 每个工具可启用/禁用，可配置参数
- 支持添加自定义 HTTP 工具（名称、描述、参数 schema、URL、headers）
- 工具调用结果在聊天面板中以卡片形式展示

### 数据库迁移

新增迁移 append 到 `migrationList()` 末尾，创建 `agent_tools` 表及索引。

---

## 两个子项目的交互

- `code_search` 和 `read_file` 工具的调用结果可以被记忆提取流程引用
- 记忆层存储的项目上下文可以为 `code_search` 提供搜索范围提示

## 交付顺序

1. 先交付持久记忆层（agent_memory）— 独立可用，无外部依赖
2. 再交付工具扩展层（agent_tool）— 依赖记忆层提供上下文增强

## 测试策略

遵循项目 TDD 规范：

- Entity：`Check()` 校验测试
- Repository：`testutils.Database(t)` + sqlmock
- Service：mockgen 生成 repo mock + `RegisterXxx` 注入
- Runtime 集成：fake provider + fake executor
- 前端：Vitest 组件测试
