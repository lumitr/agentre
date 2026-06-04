# Agent 工具扩展层 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 为内置 Agent 新增工具调用扩展能力，支持自定义工具（搜索、API、文件系统扩展等）的注册、注入和执行。

**架构：** 新增 `agent_tool` 域（entity / repo / svc / Wails binding）+ `toolext` 运行时包。工具定义注入 system prompt，工具调用在 translate 层拦截并路由到 ToolExecutor 执行。

**技术栈：** Go 1.26, cago framework, GORM/SQLite, golangci-lint, goconvey

**前置依赖：** 持久记忆层（agent_memory）已完成

---

## 文件结构

| 操作 | 文件 | 职责 |
|------|------|------|
| 创建 | `internal/model/entity/agent_tool_entity/agent_tool.go` | 充血实体 + Check() 校验 |
| 创建 | `internal/repository/agent_tool_repo/agent_tool.go` | 接口 + GORM 实现 + Register |
| 创建 | `internal/repository/agent_tool_repo/mock_agent_tool_repo/mock_agent.go` | mockgen 生成 |
| 创建 | `internal/service/agent_tool_svc/agent_tool.go` | 应用服务（CRUD + LoadTools） |
| 创建 | `internal/service/agent_tool_svc/types.go` | 请求/响应类型 |
| 创建 | `internal/pkg/agentruntime/toolext/executor.go` | ToolExecutor 接口 + ToolDef |
| 创建 | `internal/pkg/agentruntime/toolext/registry.go` | 工具注册表 |
| 创建 | `internal/pkg/agentruntime/toolext/builtin.go` | 内置执行器（web_search 等） |
| 创建 | `internal/pkg/agentruntime/toolext/http.go` | HTTP 执行器 |
| 创建 | `internal/pkg/agentruntime/toolext/prompt.go` | system prompt 生成 |
| 创建 | `internal/app/agent_tool.go` | Wails 绑定 |
| 创建 | `migrations/202606040002_agent_tools.go` | 建表迁移 |
| 修改 | `migrations/migrations.go` | 追加新迁移 |
| 修改 | `internal/bootstrap/cago.go` | 注册 agent_tool_repo + toolext |
| 修改 | `internal/pkg/agentruntime/runtimes/builtin/runtime.go` | 注入工具定义 + 拦截工具调用 |

---

### 任务 1：AgentTool 实体 + Check 校验

**文件：**
- 创建：`internal/model/entity/agent_tool_entity/agent_tool.go`

- [ ] **步骤 1：编写失败的实体测试**

```go
package agent_tool_entity_test

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"agentre/internal/model/entity/agent_tool_entity"
)

func TestAgentTool_Check(t *testing.T) {
	Convey("AgentTool.Check", t, func() {
		ctx := context.Background()

		Convey("nil receiver returns error", func() {
			var a *agent_tool_entity.AgentTool
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
		})

		Convey("empty name returns error", func() {
			a := &agent_tool_entity.AgentTool{Name: "", ExecutorType: "builtin", ParamSchema: `{"type":"object"}`}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
		})

		Convey("invalid executor_type returns error", func() {
			a := &agent_tool_entity.AgentTool{Name: "test", ExecutorType: "invalid", ParamSchema: `{"type":"object"}`}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
		})

		Convey("invalid param_schema JSON returns error", func() {
			a := &agent_tool_entity.AgentTool{Name: "test", ExecutorType: "builtin", ParamSchema: "not-json"}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
		})

		Convey("valid builtin tool passes", func() {
			a := &agent_tool_entity.AgentTool{
				Name: "web_search", ExecutorType: "builtin",
				ParamSchema: `{"type":"object","properties":{"query":{"type":"string"}}}`,
			}
			err := a.Check(ctx)
			So(err, ShouldBeNil)
		})

		Convey("valid http tool passes", func() {
			a := &agent_tool_entity.AgentTool{
				Name: "api_call", ExecutorType: "http",
				ParamSchema: `{"type":"object","properties":{"url":{"type":"string"}}}`,
				ExecutorConf: `{"url":"https://api.example.com"}`,
			}
			err := a.Check(ctx)
			So(err, ShouldBeNil)
		})
	})
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test -race ./internal/model/entity/agent_tool_entity/ -run TestAgentTool_Check -v`
预期：FAIL

- [ ] **步骤 3：编写 AgentTool 实体**

```go
// Package agent_tool_entity 维护 Agent 工具的充血实体。
package agent_tool_entity

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/cago-frame/cago/pkg/consts"
	"github.com/cago-frame/cago/pkg/i18n"

	"agentre/internal/pkg/code"
)

// ExecutorType 工具执行器类型。
type ExecutorType string

const (
	ExecutorBuiltin ExecutorType = "builtin"
	ExecutorHTTP    ExecutorType = "http"
	ExecutorMCP     ExecutorType = "mcp"
)

var validExecutorTypes = map[ExecutorType]bool{
	ExecutorBuiltin: true,
	ExecutorHTTP:    true,
	ExecutorMCP:     true,
}

// AgentTool 一条 Agent 工具配置记录。
type AgentTool struct {
	ID           int64  `gorm:"column:id;primaryKey;autoIncrement"`
	AgentID      int64  `gorm:"column:agent_id;type:bigint;not null;default:0"`
	Name         string `gorm:"column:name;type:text;not null"`
	Description  string `gorm:"column:description;type:text;not null;default:''"`
	ParamSchema  string `gorm:"column:param_schema;type:text;not null;default:'{}'"`
	ExecutorType string `gorm:"column:executor_type;type:text;not null"`
	ExecutorConf string `gorm:"column:executor_conf;type:text;not null;default:'{}'"`
	Enabled      bool   `gorm:"column:enabled;type:boolean;not null;default:true"`
	SortOrder    int    `gorm:"column:sort_order;type:int;not null;default:0"`
	Status       int    `gorm:"column:status;type:int;not null;default:1"`
	Createtime   int64  `gorm:"column:createtime;type:bigint;not null;default:0"`
	Updatetime   int64  `gorm:"column:updatetime;type:bigint;not null;default:0"`
}

func (*AgentTool) TableName() string { return "agent_tools" }

func (a *AgentTool) IsActive() bool { return a != nil && a.Status == consts.ACTIVE }

// Check 字段校验。
func (a *AgentTool) Check(ctx context.Context) error {
	if a == nil {
		return i18n.NewError(ctx, code.AgentToolNotFound)
	}
	if strings.TrimSpace(a.Name) == "" {
		return i18n.NewError(ctx, code.InvalidParameter)
	}
	if !validExecutorTypes[ExecutorType(a.ExecutorType)] {
		return i18n.NewError(ctx, code.AgentToolInvalidExecutorType)
	}
	if !isValidJSON(a.ParamSchema) {
		return i18n.NewError(ctx, code.AgentToolInvalidParamSchema)
	}
	if a.ExecutorConf != "" && a.ExecutorConf != "{}" && !isValidJSON(a.ExecutorConf) {
		return i18n.NewError(ctx, code.AgentToolInvalidExecutorConf)
	}
	return nil
}

func isValidJSON(s string) bool {
	var v json.RawMessage
	return json.Unmarshal([]byte(s), &v) == nil
}
```

- [ ] **步骤 4：添加错误码**

在 `internal/pkg/code/` 中追加：

```go
AgentToolNotFound
AgentToolInvalidExecutorType
AgentToolInvalidParamSchema
AgentToolInvalidExecutorConf
```

- [ ] **步骤 5：运行测试验证通过**

运行：`go test -race ./internal/model/entity/agent_tool_entity/ -run TestAgentTool_Check -v`
预期：PASS

- [ ] **步骤 6：Commit**

```bash
git add internal/model/entity/agent_tool_entity/ internal/pkg/code/
git commit -m "feat(agent_tool): add AgentTool entity with Check validation"
```

---

### 任务 2：数据库迁移

**文件：**
- 创建：`migrations/202606040002_agent_tools.go`
- 修改：`migrations/migrations.go`

- [ ] **步骤 1：编写迁移文件**

```go
package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// migration202606040002 建 agent_tools 表。
func migration202606040002() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202606040002",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.Exec(`CREATE TABLE IF NOT EXISTS agent_tools (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	agent_id INTEGER NOT NULL DEFAULT 0,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	param_schema TEXT NOT NULL DEFAULT '{}',
	executor_type TEXT NOT NULL,
	executor_conf TEXT NOT NULL DEFAULT '{}',
	enabled INTEGER NOT NULL DEFAULT 1,
	sort_order INTEGER NOT NULL DEFAULT 0,
	status INTEGER NOT NULL DEFAULT 1,
	createtime INTEGER NOT NULL DEFAULT 0,
	updatetime INTEGER NOT NULL DEFAULT 0
)`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_agent_tools_agent ON agent_tools(agent_id)`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uniq_agent_tools_name ON agent_tools(agent_id, name) WHERE status = 1`).Error; err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Exec(`DROP TABLE IF EXISTS agent_tools`).Error
		},
	}
}
```

- [ ] **步骤 2：追加到 migrationList()**

在 `migrations/migrations.go` 的 `migrationList()` 末尾追加：

```go
migration202606040002(), // agent_tools
```

- [ ] **步骤 3：运行迁移测试验证**

运行：`go test -race ./migrations/ -v`
预期：PASS

- [ ] **步骤 4：Commit**

```bash
git add migrations/
git commit -m "feat(agent_tool): add agent_tools table migration"
```

---

### 任务 3：AgentTool Repository

**文件：**
- 创建：`internal/repository/agent_tool_repo/agent_tool.go`

- [ ] **步骤 1：编写 repo 接口 + 实现**

```go
// Package agent_tool_repo 提供 Agent 工具的持久化访问。
package agent_tool_repo

import (
	"context"
	"errors"

	"github.com/cago-frame/cago/database/db"
	"github.com/cago-frame/cago/pkg/consts"
	"gorm.io/gorm"

	"agentre/internal/model/entity/agent_tool_entity"
)

//go:generate mockgen -source agent_tool.go -destination mock_agent_tool_repo/mock_agent.go

type AgentToolRepo interface {
	Create(ctx context.Context, t *agent_tool_entity.AgentTool) error
	Update(ctx context.Context, t *agent_tool_entity.AgentTool) error
	Delete(ctx context.Context, id int64) error
	Find(ctx context.Context, id int64) (*agent_tool_entity.AgentTool, error)
	FindByName(ctx context.Context, agentID int64, name string) (*agent_tool_entity.AgentTool, error)
	ListByAgent(ctx context.Context, agentID int64) ([]*agent_tool_entity.AgentTool, error)
	ListEnabledByAgent(ctx context.Context, agentID int64) ([]*agent_tool_entity.AgentTool, error)
	ListGlobal(ctx context.Context) ([]*agent_tool_entity.AgentTool, error)
	ListEnabledGlobal(ctx context.Context) ([]*agent_tool_entity.AgentTool, error)
}

var defaultAgentTool AgentToolRepo

func AgentTool() AgentToolRepo             { return defaultAgentTool }
func RegisterAgentTool(impl AgentToolRepo) { defaultAgentTool = impl }
func NewAgentTool() AgentToolRepo          { return &agentToolRepo{} }

type agentToolRepo struct{}

func (r *agentToolRepo) Create(ctx context.Context, t *agent_tool_entity.AgentTool) error {
	return db.Ctx(ctx).Create(t).Error
}

func (r *agentToolRepo) Update(ctx context.Context, t *agent_tool_entity.AgentTool) error {
	return db.Ctx(ctx).Save(t).Error
}

func (r *agentToolRepo) Delete(ctx context.Context, id int64) error {
	return db.Ctx(ctx).Model(&agent_tool_entity.AgentTool{}).
		Where("id = ?", id).
		Update("status", consts.DELETE).Error
}

func (r *agentToolRepo) Find(ctx context.Context, id int64) (*agent_tool_entity.AgentTool, error) {
	out := &agent_tool_entity.AgentTool{}
	err := db.Ctx(ctx).Where("id = ? AND status = ?", id, consts.ACTIVE).First(out).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *agentToolRepo) FindByName(ctx context.Context, agentID int64, name string) (*agent_tool_entity.AgentTool, error) {
	out := &agent_tool_entity.AgentTool{}
	err := db.Ctx(ctx).
		Where("agent_id = ? AND name = ? AND status = ?", agentID, name, consts.ACTIVE).
		First(out).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *agentToolRepo) ListByAgent(ctx context.Context, agentID int64) ([]*agent_tool_entity.AgentTool, error) {
	var rows []*agent_tool_entity.AgentTool
	err := db.Ctx(ctx).
		Where("agent_id = ? AND status = ?", agentID, consts.ACTIVE).
		Order("sort_order ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *agentToolRepo) ListEnabledByAgent(ctx context.Context, agentID int64) ([]*agent_tool_entity.AgentTool, error) {
	var rows []*agent_tool_entity.AgentTool
	err := db.Ctx(ctx).
		Where("agent_id = ? AND enabled = ? AND status = ?", agentID, true, consts.ACTIVE).
		Order("sort_order ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *agentToolRepo) ListGlobal(ctx context.Context) ([]*agent_tool_entity.AgentTool, error) {
	var rows []*agent_tool_entity.AgentTool
	err := db.Ctx(ctx).
		Where("agent_id = ? AND status = ?", 0, consts.ACTIVE).
		Order("sort_order ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *agentToolRepo) ListEnabledGlobal(ctx context.Context) ([]*agent_tool_entity.AgentTool, error) {
	var rows []*agent_tool_entity.AgentTool
	err := db.Ctx(ctx).
		Where("agent_id = ? AND enabled = ? AND status = ?", 0, true, consts.ACTIVE).
		Order("sort_order ASC, id ASC").
		Find(&rows).Error
	return rows, err
}
```

- [ ] **步骤 2：生成 mock**

运行：`make mock`

- [ ] **步骤 3：运行编译验证**

运行：`go build ./internal/repository/agent_tool_repo/`
预期：PASS

- [ ] **步骤 4：Commit**

```bash
git add internal/repository/agent_tool_repo/
git commit -m "feat(agent_tool): add AgentToolRepo interface and GORM implementation"
```

---

### 任务 4：toolext 包 — 接口 + 注册表 + 内置执行器

**文件：**
- 创建：`internal/pkg/agentruntime/toolext/executor.go`
- 创建：`internal/pkg/agentruntime/toolext/registry.go`
- 创建：`internal/pkg/agentruntime/toolext/builtin.go`
- 创建：`internal/pkg/agentruntime/toolext/http.go`
- 创建：`internal/pkg/agentruntime/toolext/prompt.go`

- [ ] **步骤 1：编写 ToolExecutor 接口**

```go
// Package toolext 提供 Agent 工具扩展的执行框架。
package toolext

import (
	"context"
	"encoding/json"
)

// ToolExecutor 工具执行器接口。
type ToolExecutor interface {
	Execute(ctx context.Context, params json.RawMessage) (string, error)
}

// ToolDef 工具定义，用于注入 system prompt。
type ToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	ParamSchema json.RawMessage `json:"param_schema"`
}
```

- [ ] **步骤 2：编写注册表**

```go
package toolext

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"agentre/internal/model/entity/agent_tool_entity"
	"agentre/internal/repository/agent_tool_repo"
)

var (
	mu       sync.RWMutex
	builtins = map[string]ToolExecutor{}
)

// RegisterBuiltin 注册内置执行器。init() 时调用。
func RegisterBuiltin(name string, exec ToolExecutor) {
	mu.Lock()
	defer mu.Unlock()
	builtins[name] = exec
}

// LookupBuiltin 查找内置执行器。
func LookupBuiltin(name string) (ToolExecutor, bool) {
	mu.RLock()
	defer mu.RUnlock()
	e, ok := builtins[name]
	return e, ok
}

// LoadTools 加载指定 agent 启用的工具定义（全局 + agent 专属）。
func LoadTools(ctx context.Context, agentID int64) ([]ToolDef, error) {
	var defs []ToolDef

	// 全局工具
	global, err := agent_tool_repo.AgentTool().ListEnabledGlobal(ctx)
	if err != nil {
		return nil, fmt.Errorf("toolext: load global tools: %w", err)
	}
	for _, t := range global {
		defs = append(defs, toolToDef(t))
	}

	// agent 专属工具
	agentTools, err := agent_tool_repo.AgentTool().ListEnabledByAgent(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("toolext: load agent tools: %w", err)
	}
	for _, t := range agentTools {
		defs = append(defs, toolToDef(t))
	}

	return defs, nil
}

// Execute 执行工具调用。
func Execute(ctx context.Context, toolName string, executorType string, executorConf string, params json.RawMessage) (string, error) {
	switch agent_tool_entity.ExecutorType(executorType) {
	case agent_tool_entity.ExecutorBuiltin:
		exec, ok := LookupBuiltin(toolName)
		if !ok {
			return "", fmt.Errorf("toolext: unknown builtin tool: %s", toolName)
		}
		return exec.Execute(ctx, params)
	case agent_tool_entity.ExecutorHTTP:
		return NewHTTPExecutor(executorConf).Execute(ctx, params)
	default:
		return "", fmt.Errorf("toolext: unsupported executor type: %s", executorType)
	}
}

func toolToDef(t *agent_tool_entity.AgentTool) ToolDef {
	return ToolDef{
		Name:        t.Name,
		Description: t.Description,
		ParamSchema: json.RawMessage(t.ParamSchema),
	}
}
```

- [ ] **步骤 3：编写内置执行器**

```go
package toolext

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

func init() {
	RegisterBuiltin("web_search", &webSearchExecutor{})
	RegisterBuiltin("code_search", &codeSearchExecutor{})
	RegisterBuiltin("read_file", &readFileExecutor{})
	RegisterBuiltin("list_files", &listFilesExecutor{})
}

// webSearchExecutor 搜索互联网信息。
type webSearchExecutor struct{}

type webSearchParams struct {
	Query      string `json:"query"`
	MaxResults int    `json:"max_results,omitempty"`
}

func (e *webSearchExecutor) Execute(ctx context.Context, params json.RawMessage) (string, error) {
	var p webSearchParams
	if err := json.Unmarshal(params, &p); err != nil {
		return "", fmt.Errorf("web_search: invalid params: %w", err)
	}
	if p.Query == "" {
		return "", fmt.Errorf("web_search: query is required")
	}
	if p.MaxResults <= 0 {
		p.MaxResults = 5
	}
	// 第一版：返回提示信息，实际搜索能力需要接入搜索 API。
	return fmt.Sprintf("Web search for '%s' (max %d results). [Search API integration pending]", p.Query, p.MaxResults), nil
}

// codeSearchExecutor 在项目代码库中搜索。
type codeSearchExecutor struct{}

type codeSearchParams struct {
	Query       string `json:"query"`
	FilePattern string `json:"file_pattern,omitempty"`
}

func (e *codeSearchExecutor) Execute(ctx context.Context, params json.RawMessage) (string, error) {
	var p codeSearchParams
	if err := json.Unmarshal(params, &p); err != nil {
		return "", fmt.Errorf("code_search: invalid params: %w", err)
	}
	if p.Query == "" {
		return "", fmt.Errorf("code_search: query is required")
	}
	// 使用 grep 做基础搜索
	args := []string{"-rn", "--include=*.go", "--include=*.ts", "--include=*.tsx"}
	if p.FilePattern != "" {
		args = append(args, fmt.Sprintf("--include=%s", p.FilePattern))
	}
	args = append(args, p.Query, ".")
	cmd := exec.CommandContext(ctx, "grep", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(err.Error(), "exit status 1") {
			return "No results found.", nil
		}
		return "", fmt.Errorf("code_search: %w", err)
	}
	// 截取前 2000 字符
	result := string(out)
	if len(result) > 2000 {
		result = result[:2000] + "\n... (truncated)"
	}
	return result, nil
}

// readFileExecutor 读取项目文件内容。
type readFileExecutor struct{}

type readFileParams struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
}

func (e *readFileExecutor) Execute(ctx context.Context, params json.RawMessage) (string, error) {
	var p readFileParams
	if err := json.Unmarshal(params, &p); err != nil {
		return "", fmt.Errorf("read_file: invalid params: %w", err)
	}
	if p.Path == "" {
		return "", fmt.Errorf("read_file: path is required")
	}
	// 安全检查：不允许读取上层目录
	if strings.Contains(p.Path, "..") {
		return "", fmt.Errorf("read_file: path traversal not allowed")
	}
	cmd := exec.CommandContext(ctx, "cat", "-n", p.Path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("read_file: %w", err)
	}
	return string(out), nil
}

// listFilesExecutor 列出目录结构。
type listFilesExecutor struct{}

type listFilesParams struct {
	Path    string `json:"path"`
	Pattern string `json:"pattern,omitempty"`
}

func (e *listFilesExecutor) Execute(ctx context.Context, params json.RawMessage) (string, error) {
	var p listFilesParams
	if err := json.Unmarshal(params, &p); err != nil {
		return "", fmt.Errorf("list_files: invalid params: %w", err)
	}
	if p.Path == "" {
		p.Path = "."
	}
	args := []string{"-L", "3", p.Path}
	if p.Pattern != "" {
		args = append([]string{"-I", p.Pattern}, args...)
	}
	cmd := exec.CommandContext(ctx, "tree", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// tree 可能未安装，回退到 ls
		cmd = exec.CommandContext(ctx, "ls", "-la", p.Path)
		out, err = cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("list_files: %w", err)
		}
	}
	return string(out), nil
}
```

- [ ] **步骤 4：编写 HTTP 执行器**

```go
package toolext

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPExecutor 调用外部 HTTP API。
type HTTPExecutor struct {
	URL     string            `json:"url"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Timeout int               `json:"timeout,omitempty"`
}

func NewHTTPExecutor(conf string) *HTTPExecutor {
	var e HTTPExecutor
	_ = json.Unmarshal([]byte(conf), &e)
	if e.Method == "" {
		e.Method = "POST"
	}
	if e.Timeout <= 0 {
		e.Timeout = 30
	}
	return &e
}

func (e *HTTPExecutor) Execute(ctx context.Context, params json.RawMessage) (string, error) {
	if e.URL == "" {
		return "", fmt.Errorf("http_executor: url is required")
	}
	timeout := time.Duration(e.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, e.Method, e.URL, bytes.NewReader(params))
	if err != nil {
		return "", fmt.Errorf("http_executor: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range e.Headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http_executor: do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", fmt.Errorf("http_executor: read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("http_executor: status %d: %s", resp.StatusCode, string(body))
	}
	return string(body), nil
}
```

- [ ] **步骤 5：编写 prompt 生成**

```go
package toolext

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FormatToolsPrompt 将工具定义格式化为 system prompt 片段。
func FormatToolsPrompt(defs []ToolDef) string {
	if len(defs) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n\n[Available Tools]\n")
	sb.WriteString("You have access to the following custom tools. Use them when appropriate:\n\n")
	for _, d := range defs {
		sb.WriteString(fmt.Sprintf("### %s\n%s\n\nParameters:\n```json\n%s\n```\n\n", d.Name, d.Description, indentJSON(d.ParamSchema)))
	}
	sb.WriteString("To use a tool, include a tool_use block with the tool name and parameters in your response.\n")
	return sb.String()
}

func indentJSON(raw json.RawMessage) string {
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return string(raw)
	}
	return string(pretty)
}
```

- [ ] **步骤 6：运行编译验证**

运行：`go build ./internal/pkg/agentruntime/toolext/`
预期：PASS

- [ ] **步骤 7：Commit**

```bash
git add internal/pkg/agentruntime/toolext/
git commit -m "feat(agent_tool): add toolext package with executor, registry, and builtin tools"
```

---

### 任务 5：AgentTool Service

**文件：**
- 创建：`internal/service/agent_tool_svc/types.go`
- 创建：`internal/service/agent_tool_svc/agent_tool.go`

- [ ] **步骤 1：编写请求/响应类型**

```go
// Package agent_tool_svc 暴露 Agent 工具的应用服务接口与请求/响应类型。
package agent_tool_svc

// ToolItem 工具条目 DTO。
type ToolItem struct {
	ID           int64  `json:"id"`
	AgentID      int64  `json:"agentId"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	ParamSchema  string `json:"paramSchema"`
	ExecutorType string `json:"executorType"`
	ExecutorConf string `json:"executorConf"`
	Enabled      bool   `json:"enabled"`
	SortOrder    int    `json:"sortOrder"`
	Createtime   int64  `json:"createtime"`
	Updatetime   int64  `json:"updatetime"`
}

// CreateToolRequest 新建工具。
type CreateToolRequest struct {
	AgentID      int64  `json:"agentId"`
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	ParamSchema  string `json:"paramSchema"`
	ExecutorType string `json:"executorType" binding:"required"`
	ExecutorConf string `json:"executorConf"`
}

type CreateToolResponse struct {
	Item *ToolItem `json:"item"`
}

// UpdateToolRequest 更新工具。
type UpdateToolRequest struct {
	ID           int64  `json:"id" binding:"required"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	ParamSchema  string `json:"paramSchema"`
	ExecutorType string `json:"executorType"`
	ExecutorConf string `json:"executorConf"`
	Enabled      *bool  `json:"enabled"`
}

type UpdateToolResponse struct {
	Item *ToolItem `json:"item"`
}

// DeleteToolRequest 删除工具。
type DeleteToolRequest struct {
	ID int64 `json:"id" binding:"required"`
}

type DeleteToolResponse struct{}

// ListToolsRequest 列出工具。
type ListToolsRequest struct {
	AgentID int64 `json:"agentId"`
}

type ListToolsResponse struct {
	Items []*ToolItem `json:"items"`
}
```

- [ ] **步骤 2：编写 service 接口 + 实现**

```go
package agent_tool_svc

import (
	"context"
	"strings"
	"time"

	"github.com/cago-frame/cago/pkg/i18n"

	"agentre/internal/model/entity/agent_tool_entity"
	"agentre/internal/pkg/code"
	"agentre/internal/repository/agent_tool_repo"
)

type AgentToolSvc interface {
	Create(ctx context.Context, req *CreateToolRequest) (*CreateToolResponse, error)
	Update(ctx context.Context, req *UpdateToolRequest) (*UpdateToolResponse, error)
	Delete(ctx context.Context, req *DeleteToolRequest) (*DeleteToolResponse, error)
	List(ctx context.Context, req *ListToolsRequest) (*ListToolsResponse, error)
}

type agentToolSvc struct {
	now func() int64
}

var defaultAgentTool AgentToolSvc = &agentToolSvc{now: func() int64 { return time.Now().Unix() }}

func AgentTool() AgentToolSvc { return defaultAgentTool }

func (s *agentToolSvc) Create(ctx context.Context, req *CreateToolRequest) (*CreateToolResponse, error) {
	now := s.now()
	t := &agent_tool_entity.AgentTool{
		AgentID:      req.AgentID,
		Name:         strings.TrimSpace(req.Name),
		Description:  strings.TrimSpace(req.Description),
		ParamSchema:  req.ParamSchema,
		ExecutorType: strings.TrimSpace(req.ExecutorType),
		ExecutorConf: req.ExecutorConf,
		Enabled:      true,
		Status:       1,
		Createtime:   now,
		Updatetime:   now,
	}
	if t.ParamSchema == "" {
		t.ParamSchema = "{}"
	}
	if t.ExecutorConf == "" {
		t.ExecutorConf = "{}"
	}
	if err := t.Check(ctx); err != nil {
		return nil, err
	}
	if err := agent_tool_repo.AgentTool().Create(ctx, t); err != nil {
		return nil, err
	}
	return &CreateToolResponse{Item: toItem(t)}, nil
}

func (s *agentToolSvc) Update(ctx context.Context, req *UpdateToolRequest) (*UpdateToolResponse, error) {
	existing, err := agent_tool_repo.AgentTool().Find(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, i18n.NewError(ctx, code.AgentToolNotFound)
	}
	if req.Name != "" {
		existing.Name = strings.TrimSpace(req.Name)
	}
	if req.Description != "" {
		existing.Description = strings.TrimSpace(req.Description)
	}
	if req.ParamSchema != "" {
		existing.ParamSchema = req.ParamSchema
	}
	if req.ExecutorType != "" {
		existing.ExecutorType = strings.TrimSpace(req.ExecutorType)
	}
	if req.ExecutorConf != "" {
		existing.ExecutorConf = req.ExecutorConf
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	existing.Updatetime = s.now()
	if err := existing.Check(ctx); err != nil {
		return nil, err
	}
	if err := agent_tool_repo.AgentTool().Update(ctx, existing); err != nil {
		return nil, err
	}
	return &UpdateToolResponse{Item: toItem(existing)}, nil
}

func (s *agentToolSvc) Delete(ctx context.Context, req *DeleteToolRequest) (*DeleteToolResponse, error) {
	existing, err := agent_tool_repo.AgentTool().Find(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, i18n.NewError(ctx, code.AgentToolNotFound)
	}
	if err := agent_tool_repo.AgentTool().Delete(ctx, existing.ID); err != nil {
		return nil, err
	}
	return &DeleteToolResponse{}, nil
}

func (s *agentToolSvc) List(ctx context.Context, req *ListToolsRequest) (*ListToolsResponse, error) {
	var rows []*agent_tool_entity.AgentTool
	var err error
	if req.AgentID > 0 {
		rows, err = agent_tool_repo.AgentTool().ListByAgent(ctx, req.AgentID)
	} else {
		rows, err = agent_tool_repo.AgentTool().ListGlobal(ctx)
	}
	if err != nil {
		return nil, err
	}
	items := make([]*ToolItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, toItem(r))
	}
	return &ListToolsResponse{Items: items}, nil
}

func toItem(t *agent_tool_entity.AgentTool) *ToolItem {
	return &ToolItem{
		ID:           t.ID,
		AgentID:      t.AgentID,
		Name:         t.Name,
		Description:  t.Description,
		ParamSchema:  t.ParamSchema,
		ExecutorType: t.ExecutorType,
		ExecutorConf: t.ExecutorConf,
		Enabled:      t.Enabled,
		SortOrder:    t.SortOrder,
		Createtime:   t.Createtime,
		Updatetime:   t.Updatetime,
	}
}
```

- [ ] **步骤 3：运行编译验证**

运行：`go build ./internal/service/agent_tool_svc/`
预期：PASS

- [ ] **步骤 4：Commit**

```bash
git add internal/service/agent_tool_svc/
git commit -m "feat(agent_tool): add AgentToolSvc CRUD operations"
```

---

### 任务 6：Wails 绑定 + Bootstrap 注册

**文件：**
- 创建：`internal/app/agent_tool.go`
- 修改：`internal/bootstrap/cago.go`

- [ ] **步骤 1：编写 Wails 绑定**

```go
package app

import (
	"agentre/internal/service/agent_tool_svc"
)

// CreateAgentTool 新建 Agent 工具。
func (a *App) CreateAgentTool(req *agent_tool_svc.CreateToolRequest) (*agent_tool_svc.CreateToolResponse, error) {
	return agent_tool_svc.AgentTool().Create(a.ctx, req)
}

// UpdateAgentTool 更新 Agent 工具。
func (a *App) UpdateAgentTool(req *agent_tool_svc.UpdateToolRequest) (*agent_tool_svc.UpdateToolResponse, error) {
	return agent_tool_svc.AgentTool().Update(a.ctx, req)
}

// DeleteAgentTool 删除 Agent 工具。
func (a *App) DeleteAgentTool(req *agent_tool_svc.DeleteToolRequest) (*agent_tool_svc.DeleteToolResponse, error) {
	return agent_tool_svc.AgentTool().Delete(a.ctx, req)
}

// ListAgentTools 列出 Agent 工具。
func (a *App) ListAgentTools(req *agent_tool_svc.ListToolsRequest) (*agent_tool_svc.ListToolsResponse, error) {
	return agent_tool_svc.AgentTool().List(a.ctx, req)
}
```

- [ ] **步骤 2：在 cago.go 中注册**

添加 import：
```go
"agentre/internal/repository/agent_tool_repo"
```

在 `Init()` 追加：
```go
agent_tool_repo.RegisterAgentTool(agent_tool_repo.NewAgentTool())
```

- [ ] **步骤 3：运行编译验证**

运行：`go build ./internal/app/ ./internal/bootstrap/`
预期：PASS

- [ ] **步骤 4：Commit**

```bash
git add internal/app/agent_tool.go internal/bootstrap/cago.go
git commit -m "feat(agent_tool): add Wails bindings and bootstrap registration"
```

---

### 任务 7：builtin runtime 集成工具扩展

**文件：**
- 修改：`internal/pkg/agentruntime/runtimes/builtin/runtime.go`
- 修改：`internal/pkg/agentruntime/runtimes/builtin/translator.go`

- [ ] **步骤 1：在 Run() 中注入工具定义**

在 `runtime.go` 的 `Run()` 方法中，记忆注入之后，追加工具定义注入：

```go
// 加载工具定义
toolDefs, toolErr := toolext.LoadTools(ctx, req.AgentID)
if toolErr != nil {
    logger.Ctx(ctx).Warn("builtin runtime: load tools failed", zap.Error(toolErr))
}
if len(toolDefs) > 0 {
    sys = sys + toolext.FormatToolsPrompt(toolDefs)
}
```

- [ ] **步骤 2：在 translate 中拦截自定义工具调用**

在 `translator.go` 的 `translate()` 函数中，对 `EventPreToolUse` 事件增加自定义工具检测：

```go
case agent.EventPreToolUse:
    if ev.Tool == nil {
        return nil
    }
    // 检查是否为自定义工具
    if isCustomTool(ev.Tool.Name) {
        return []agentruntime.Event{agentruntime.ToolCall{
            ID:    ev.Tool.ToolUseID,
            Name:  ev.Tool.Name,
            Input: marshalToolInput(ev.Tool.Input),
        }}
    }
    // 原有逻辑
    return []agentruntime.Event{agentruntime.ToolCall{...}}
```

注意：自定义工具的实际执行需要在 cago 工具循环中拦截。具体实现方式取决于 cago 框架是否支持自定义工具注册。如果不支持，需要在 Run() 的事件循环中拦截并执行，然后通过 Steer 注入结果。此步骤需要在实现时验证 cago 框架的能力。

- [ ] **步骤 3：运行编译验证**

运行：`go build ./internal/pkg/agentruntime/runtimes/builtin/`
预期：PASS

- [ ] **步骤 4：Commit**

```bash
git add internal/pkg/agentruntime/runtimes/builtin/
git commit -m "feat(agent_tool): integrate tool extension into builtin runtime"
```

---

### 任务 8：前端工具标签页

**文件：**
- 运行 `make generate` 生成前端绑定
- 修改：前端 Agent 设置页，添加工具标签页
- 修改：`frontend/src/i18n/locales/{zh-CN,en}/common.json`

- [ ] **步骤 1：生成 Wails 绑定**

运行：`make generate`

- [ ] **步骤 2：添加前端工具标签页组件**

在 Agent 设置/编辑页中添加"工具"标签页，调用 `ListAgentTools` 展示列表，支持 `CreateAgentTool` / `UpdateAgentTool` / `DeleteAgentTool` 操作。

- [ ] **步骤 3：添加 i18n 文案**

在 common.json 中添加工具相关的 UI 文案。

- [ ] **步骤 4：运行前端测试**

运行：`cd frontend && pnpm test`
预期：PASS

- [ ] **步骤 5：Commit**

```bash
git add frontend/
git commit -m "feat(agent_tool): add frontend tools tab and i18n"
```

---

### 任务 9：端到端验证

- [ ] **步骤 1：运行完整后端测试**

运行：`make test-backend`
预期：PASS

- [ ] **步骤 2：运行 lint**

运行：`make lint`
预期：PASS

- [ ] **步骤 3：运行前端测试**

运行：`make test-frontend`
预期：PASS

- [ ] **步骤 4：手动验证**

启动 `make dev`，在 UI 中：
1. 打开 Agent 设置页，切换到"工具"标签页
2. 启用内置工具（如 code_search）
3. 开始新聊天会话，验证工具定义被注入 system prompt
4. 让 Agent 尝试调用工具，验证执行流程
