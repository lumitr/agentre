# Agent 持久记忆层 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 为内置 Agent 新增持久记忆能力，支持跨会话的用户偏好、项目上下文和对话摘要的存储、注入与自动提取。

**架构：** 新增 `agent_memory` 域（entity / repo / svc / Wails binding），遵循项目现有分层模式。builtin runtime 在 `Run()` 时通过 `coding.AppendSystem()` 注入记忆；chat_svc 在 turn 结束时触发异步记忆提取。

**技术栈：** Go 1.26, cago framework, GORM/SQLite, golangci-lint, goconvey

---

## 文件结构

| 操作 | 文件 | 职责 |
|------|------|------|
| 创建 | `internal/model/entity/agent_memory_entity/agent_memory.go` | 充血实体 + Check() 校验 |
| 创建 | `internal/repository/agent_memory_repo/agent_memory.go` | 接口 + GORM 实现 + Register |
| 创建 | `internal/repository/agent_memory_repo/mock_agent_memory_repo/mock_agent.go` | mockgen 生成 |
| 创建 | `internal/service/agent_memory_svc/agent_memory.go` | 应用服务（CRUD + LoadMemories + ExtractFromSession） |
| 创建 | `internal/service/agent_memory_svc/types.go` | 请求/响应类型 |
| 创建 | `internal/service/agent_memory_svc/agent_memory_test.go` | 服务单测 |
| 创建 | `internal/service/agent_memory_svc/agent_memory_internal_test.go` | 内部测试 |
| 创建 | `internal/app/agent_memory.go` | Wails 绑定 |
| 创建 | `migrations/202606040001_agent_memories.go` | 建表迁移 |
| 修改 | `migrations/migrations.go` | 追加新迁移到 migrationList() |
| 修改 | `internal/bootstrap/cago.go` | 注册 agent_memory_repo + agent_memory_svc |
| 修改 | `internal/pkg/agentruntime/runtimes/builtin/runtime.go` | Run() 中注入记忆 |
| 修改 | `internal/service/chat_svc/chat.go` | turn 结束时触发 ExtractFromSession |

---

### 任务 1：AgentMemory 实体 + Check 校验

**文件：**
- 创建：`internal/model/entity/agent_memory_entity/agent_memory.go`
- 测试：同文件 `TestAgentMemory_Check`（entity 测试内联）

- [ ] **步骤 1：编写失败的实体测试**

```go
package agent_memory_entity_test

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"agentre/internal/model/entity/agent_memory_entity"
)

func TestAgentMemory_Check(t *testing.T) {
	Convey("AgentMemory.Check", t, func() {
		ctx := context.Background()

		Convey("nil receiver returns error", func() {
			var a *agent_memory_entity.AgentMemory
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
		})

		Convey("empty agent_id returns error", func() {
			a := &agent_memory_entity.AgentMemory{AgentID: 0, Scope: "user", Category: "preference", Content: "test"}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
		})

		Convey("invalid scope returns error", func() {
			a := &agent_memory_entity.AgentMemory{AgentID: 1, Scope: "invalid", Category: "preference", Content: "test"}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
		})

		Convey("invalid category returns error", func() {
			a := &agent_memory_entity.AgentMemory{AgentID: 1, Scope: "user", Category: "invalid", Content: "test"}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
		})

		Convey("empty content returns error", func() {
			a := &agent_memory_entity.AgentMemory{AgentID: 1, Scope: "user", Category: "preference", Content: ""}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
		})

		Convey("session scope without session_id returns error", func() {
			a := &agent_memory_entity.AgentMemory{AgentID: 1, Scope: "session", Category: "summary", Content: "test", SessionID: 0}
			err := a.Check(ctx)
			So(err, ShouldNotBeNil)
		})

		Convey("valid user preference passes", func() {
			a := &agent_memory_entity.AgentMemory{AgentID: 1, Scope: "user", Category: "preference", Content: "likes TypeScript", Key: "language"}
			err := a.Check(ctx)
			So(err, ShouldBeNil)
		})

		Convey("valid session summary passes", func() {
			a := &agent_memory_entity.AgentMemory{AgentID: 1, Scope: "session", Category: "summary", Content: "discussed API design", SessionID: 42}
			err := a.Check(ctx)
			So(err, ShouldBeNil)
		})
	})
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test -race ./internal/model/entity/agent_memory_entity/ -run TestAgentMemory_Check -v`
预期：FAIL，包不存在

- [ ] **步骤 3：编写 AgentMemory 实体**

```go
// Package agent_memory_entity 维护 Agent 记忆的充血实体。
package agent_memory_entity

import (
	"context"
	"strings"

	"github.com/cago-frame/cago/pkg/consts"
	"github.com/cago-frame/cago/pkg/i18n"

	"agentre/internal/pkg/code"
)

// Scope 记忆作用域。
type Scope string

const (
	ScopeUser    Scope = "user"
	ScopeProject Scope = "project"
	ScopeSession Scope = "session"
)

// Category 记忆类别。
type Category string

const (
	CategoryPreference Category = "preference"
	CategorySummary    Category = "summary"
	CategoryContext    Category = "context"
	CategoryCustom     Category = "custom"
)

// Source 记忆来源。
type Source string

const (
	SourceAuto   Source = "auto"
	SourceManual Source = "manual"
)

var validScopes = map[Scope]bool{
	ScopeUser:    true,
	ScopeProject: true,
	ScopeSession: true,
}

var validCategories = map[Category]bool{
	CategoryPreference: true,
	CategorySummary:    true,
	CategoryContext:    true,
	CategoryCustom:     true,
}

// AgentMemory 一条 Agent 记忆记录。
type AgentMemory struct {
	ID         int64  `gorm:"column:id;primaryKey;autoIncrement"`
	AgentID    int64  `gorm:"column:agent_id;type:bigint;not null"`
	Scope      string `gorm:"column:scope;type:text;not null"`
	SessionID  int64  `gorm:"column:session_id;type:bigint;not null;default:0"`
	Category   string `gorm:"column:category;type:text;not null"`
	Key        string `gorm:"column:key;type:text;not null;default:''"`
	Content    string `gorm:"column:content;type:text;not null"`
	Source     string `gorm:"column:source;type:text;not null;default:'manual'"`
	Status     int    `gorm:"column:status;type:int;not null;default:1"`
	Createtime int64  `gorm:"column:createtime;type:bigint;not null;default:0"`
	Updatetime int64  `gorm:"column:updatetime;type:bigint;not null;default:0"`
}

func (*AgentMemory) TableName() string { return "agent_memories" }

func (a *AgentMemory) IsActive() bool { return a != nil && a.Status == consts.ACTIVE }

// Check 字段校验。
func (a *AgentMemory) Check(ctx context.Context) error {
	if a == nil {
		return i18n.NewError(ctx, code.AgentMemoryNotFound)
	}
	if a.AgentID <= 0 {
		return i18n.NewError(ctx, code.InvalidParameter)
	}
	if !validScopes[Scope(a.Scope)] {
		return i18n.NewError(ctx, code.AgentMemoryInvalidScope)
	}
	if !validCategories[Category(a.Category)] {
		return i18n.NewError(ctx, code.AgentMemoryInvalidCategory)
	}
	if strings.TrimSpace(a.Content) == "" {
		return i18n.NewError(ctx, code.AgentMemoryEmptyContent)
	}
	if Scope(a.Scope) == ScopeSession && a.SessionID <= 0 {
		return i18n.NewError(ctx, code.AgentMemorySessionRequired)
	}
	return nil
}
```

- [ ] **步骤 4：添加错误码**

在 `internal/pkg/code/` 中添加错误码常量：

```go
AgentMemoryNotFound          = 40001 + iota // 根据实际 code 包的下一个可用值
AgentMemoryInvalidScope
AgentMemoryInvalidCategory
AgentMemoryEmptyContent
AgentMemorySessionRequired
```

- [ ] **步骤 5：运行测试验证通过**

运行：`go test -race ./internal/model/entity/agent_memory_entity/ -run TestAgentMemory_Check -v`
预期：PASS

- [ ] **步骤 6：Commit**

```bash
git add internal/model/entity/agent_memory_entity/ internal/pkg/code/
git commit -m "feat(agent_memory): add AgentMemory entity with Check validation"
```

---

### 任务 2：数据库迁移

**文件：**
- 创建：`migrations/202606040001_agent_memories.go`
- 修改：`migrations/migrations.go`

- [ ] **步骤 1：编写迁移文件**

```go
package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// migration202606040001 建 agent_memories 表。
func migration202606040001() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202606040001",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.Exec(`CREATE TABLE IF NOT EXISTS agent_memories (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	agent_id INTEGER NOT NULL,
	scope TEXT NOT NULL,
	session_id INTEGER NOT NULL DEFAULT 0,
	category TEXT NOT NULL,
	key TEXT NOT NULL DEFAULT '',
	content TEXT NOT NULL,
	source TEXT NOT NULL DEFAULT 'manual',
	status INTEGER NOT NULL DEFAULT 1,
	createtime INTEGER NOT NULL DEFAULT 0,
	updatetime INTEGER NOT NULL DEFAULT 0
)`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_agent_memories_agent_scope ON agent_memories(agent_id, scope)`).Error; err != nil {
				return err
			}
			if err := tx.Exec(`CREATE INDEX IF NOT EXISTS idx_agent_memories_session ON agent_memories(session_id) WHERE scope = 'session'`).Error; err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Exec(`DROP TABLE IF EXISTS agent_memories`).Error
		},
	}
}
```

- [ ] **步骤 2：追加到 migrationList()**

在 `migrations/migrations.go` 的 `migrationList()` 末尾追加：

```go
migration202606040001(), // agent_memories
```

- [ ] **步骤 3：运行迁移测试验证**

运行：`go test -race ./migrations/ -v`
预期：PASS

- [ ] **步骤 4：Commit**

```bash
git add migrations/
git commit -m "feat(agent_memory): add agent_memories table migration"
```

---

### 任务 3：AgentMemory Repository

**文件：**
- 创建：`internal/repository/agent_memory_repo/agent_memory.go`
- 创建：`internal/repository/agent_memory_repo/mock_agent_memory_repo/mock_agent.go`（mockgen 生成）

- [ ] **步骤 1：编写 repo 接口 + 实现**

```go
// Package agent_memory_repo 提供 Agent 记忆的持久化访问。
package agent_memory_repo

import (
	"context"
	"errors"

	"github.com/cago-frame/cago/database/db"
	"github.com/cago-frame/cago/pkg/consts"
	"gorm.io/gorm"

	"agentre/internal/model/entity/agent_memory_entity"
)

//go:generate mockgen -source agent_memory.go -destination mock_agent_memory_repo/mock_agent.go

type AgentMemoryRepo interface {
	Create(ctx context.Context, m *agent_memory_entity.AgentMemory) error
	Update(ctx context.Context, m *agent_memory_entity.AgentMemory) error
	Delete(ctx context.Context, id int64) error
	Find(ctx context.Context, id int64) (*agent_memory_entity.AgentMemory, error)
	ListByAgent(ctx context.Context, agentID int64) ([]*agent_memory_entity.AgentMemory, error)
	ListByAgentAndScope(ctx context.Context, agentID int64, scope string) ([]*agent_memory_entity.AgentMemory, error)
	FindByAgentAndKey(ctx context.Context, agentID int64, scope, key string) (*agent_memory_entity.AgentMemory, error)
	FindBySession(ctx context.Context, sessionID int64) ([]*agent_memory_entity.AgentMemory, error)
	UpsertByAgentAndKey(ctx context.Context, m *agent_memory_entity.AgentMemory) error
}

var defaultAgentMemory AgentMemoryRepo

func AgentMemory() AgentMemoryRepo                 { return defaultAgentMemory }
func RegisterAgentMemory(impl AgentMemoryRepo)     { defaultAgentMemory = impl }
func NewAgentMemory() AgentMemoryRepo              { return &agentMemoryRepo{} }

type agentMemoryRepo struct{}

func (r *agentMemoryRepo) Create(ctx context.Context, m *agent_memory_entity.AgentMemory) error {
	return db.Ctx(ctx).Create(m).Error
}

func (r *agentMemoryRepo) Update(ctx context.Context, m *agent_memory_entity.AgentMemory) error {
	return db.Ctx(ctx).Save(m).Error
}

func (r *agentMemoryRepo) Delete(ctx context.Context, id int64) error {
	return db.Ctx(ctx).Model(&agent_memory_entity.AgentMemory{}).
		Where("id = ?", id).
		Update("status", consts.DELETE).Error
}

func (r *agentMemoryRepo) Find(ctx context.Context, id int64) (*agent_memory_entity.AgentMemory, error) {
	out := &agent_memory_entity.AgentMemory{}
	err := db.Ctx(ctx).Where("id = ? AND status = ?", id, consts.ACTIVE).First(out).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *agentMemoryRepo) ListByAgent(ctx context.Context, agentID int64) ([]*agent_memory_entity.AgentMemory, error) {
	var rows []*agent_memory_entity.AgentMemory
	err := db.Ctx(ctx).
		Where("agent_id = ? AND status = ?", agentID, consts.ACTIVE).
		Order("scope ASC, category ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *agentMemoryRepo) ListByAgentAndScope(ctx context.Context, agentID int64, scope string) ([]*agent_memory_entity.AgentMemory, error) {
	var rows []*agent_memory_entity.AgentMemory
	err := db.Ctx(ctx).
		Where("agent_id = ? AND scope = ? AND status = ?", agentID, scope, consts.ACTIVE).
		Order("category ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *agentMemoryRepo) FindByAgentAndKey(ctx context.Context, agentID int64, scope, key string) (*agent_memory_entity.AgentMemory, error) {
	out := &agent_memory_entity.AgentMemory{}
	err := db.Ctx(ctx).
		Where("agent_id = ? AND scope = ? AND key = ? AND status = ?", agentID, scope, key, consts.ACTIVE).
		First(out).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *agentMemoryRepo) FindBySession(ctx context.Context, sessionID int64) ([]*agent_memory_entity.AgentMemory, error) {
	var rows []*agent_memory_entity.AgentMemory
	err := db.Ctx(ctx).
		Where("session_id = ? AND scope = ? AND status = ?", sessionID, "session", consts.ACTIVE).
		Find(&rows).Error
	return rows, err
}

func (r *agentMemoryRepo) UpsertByAgentAndKey(ctx context.Context, m *agent_memory_entity.AgentMemory) error {
	existing, err := r.FindByAgentAndKey(ctx, m.AgentID, m.Scope, m.Key)
	if err != nil {
		return err
	}
	if existing != nil {
		existing.Content = m.Content
		existing.Category = m.Category
		existing.Source = m.Source
		existing.Updatetime = m.Updatetime
		return r.Update(ctx, existing)
	}
	return r.Create(ctx, m)
}
```

- [ ] **步骤 2：生成 mock**

运行：`make mock`

- [ ] **步骤 3：运行编译验证**

运行：`go build ./internal/repository/agent_memory_repo/`
预期：PASS

- [ ] **步骤 4：Commit**

```bash
git add internal/repository/agent_memory_repo/
git commit -m "feat(agent_memory): add AgentMemoryRepo interface and GORM implementation"
```

---

### 任务 4：AgentMemory Service — CRUD

**文件：**
- 创建：`internal/service/agent_memory_svc/types.go`
- 创建：`internal/service/agent_memory_svc/agent_memory.go`

- [ ] **步骤 1：编写请求/响应类型**

```go
// Package agent_memory_svc 暴露 Agent 记忆的应用服务接口与请求/响应类型。
package agent_memory_svc

// MemoryItem 记忆条目 DTO。
type MemoryItem struct {
	ID         int64  `json:"id"`
	AgentID    int64  `json:"agentId"`
	Scope      string `json:"scope"`
	SessionID  int64  `json:"sessionId"`
	Category   string `json:"category"`
	Key        string `json:"key"`
	Content    string `json:"content"`
	Source     string `json:"source"`
	Createtime int64  `json:"createtime"`
	Updatetime int64  `json:"updatetime"`
}

// CreateMemoryRequest 新建记忆。
type CreateMemoryRequest struct {
	AgentID  int64  `json:"agentId" binding:"required"`
	Scope    string `json:"scope" binding:"required"`
	SessionID int64 `json:"sessionId"`
	Category string `json:"category" binding:"required"`
	Key      string `json:"key"`
	Content  string `json:"content" binding:"required"`
}

type CreateMemoryResponse struct {
	Item *MemoryItem `json:"item"`
}

// UpdateMemoryRequest 更新记忆。
type UpdateMemoryRequest struct {
	ID       int64  `json:"id" binding:"required"`
	Content  string `json:"content" binding:"required"`
	Category string `json:"category"`
	Key      string `json:"key"`
}

type UpdateMemoryResponse struct {
	Item *MemoryItem `json:"item"`
}

// DeleteMemoryRequest 删除记忆。
type DeleteMemoryRequest struct {
	ID int64 `json:"id" binding:"required"`
}

type DeleteMemoryResponse struct{}

// ListMemoryRequest 列出记忆。
type ListMemoryRequest struct {
	AgentID int64  `json:"agentId" binding:"required"`
	Scope   string `json:"scope"`
}

type ListMemoryResponse struct {
	Items []*MemoryItem `json:"items"`
}
```

- [ ] **步骤 2：编写 service 接口 + CRUD 实现**

```go
package agent_memory_svc

import (
	"context"
	"strings"
	"time"

	"github.com/cago-frame/cago/pkg/i18n"

	"agentre/internal/model/entity/agent_memory_entity"
	"agentre/internal/pkg/code"
	"agentre/internal/repository/agent_memory_repo"
)

type AgentMemorySvc interface {
	Create(ctx context.Context, req *CreateMemoryRequest) (*CreateMemoryResponse, error)
	Update(ctx context.Context, req *UpdateMemoryRequest) (*UpdateMemoryResponse, error)
	Delete(ctx context.Context, req *DeleteMemoryRequest) (*DeleteMemoryResponse, error)
	List(ctx context.Context, req *ListMemoryRequest) (*ListMemoryResponse, error)
	LoadMemories(ctx context.Context, agentID int64, scope string, sessionID int64) ([]*MemoryItem, error)
	ExtractFromSession(ctx context.Context, sessionID int64) error
}

type agentMemorySvc struct {
	now func() int64
}

var defaultAgentMemory AgentMemorySvc = &agentMemorySvc{now: func() int64 { return time.Now().Unix() }}

func AgentMemory() AgentMemorySvc { return defaultAgentMemory }

func (s *agentMemorySvc) Create(ctx context.Context, req *CreateMemoryRequest) (*CreateMemoryResponse, error) {
	now := s.now()
	m := &agent_memory_entity.AgentMemory{
		AgentID:   req.AgentID,
		Scope:     strings.TrimSpace(req.Scope),
		SessionID: req.SessionID,
		Category:  strings.TrimSpace(req.Category),
		Key:       strings.TrimSpace(req.Key),
		Content:   strings.TrimSpace(req.Content),
		Source:    string(agent_memory_entity.SourceManual),
		Status:    1,
		Createtime: now,
		Updatetime: now,
	}
	if err := m.Check(ctx); err != nil {
		return nil, err
	}
	if err := agent_memory_repo.AgentMemory().Create(ctx, m); err != nil {
		return nil, err
	}
	return &CreateMemoryResponse{Item: toItem(m)}, nil
}

func (s *agentMemorySvc) Update(ctx context.Context, req *UpdateMemoryRequest) (*UpdateMemoryResponse, error) {
	existing, err := agent_memory_repo.AgentMemory().Find(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, i18n.NewError(ctx, code.AgentMemoryNotFound)
	}
	existing.Content = strings.TrimSpace(req.Content)
	if req.Category != "" {
		existing.Category = strings.TrimSpace(req.Category)
	}
	if req.Key != "" {
		existing.Key = strings.TrimSpace(req.Key)
	}
	existing.Updatetime = s.now()
	if err := existing.Check(ctx); err != nil {
		return nil, err
	}
	if err := agent_memory_repo.AgentMemory().Update(ctx, existing); err != nil {
		return nil, err
	}
	return &UpdateMemoryResponse{Item: toItem(existing)}, nil
}

func (s *agentMemorySvc) Delete(ctx context.Context, req *DeleteMemoryRequest) (*DeleteMemoryResponse, error) {
	existing, err := agent_memory_repo.AgentMemory().Find(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, i18n.NewError(ctx, code.AgentMemoryNotFound)
	}
	if err := agent_memory_repo.AgentMemory().Delete(ctx, existing.ID); err != nil {
		return nil, err
	}
	return &DeleteMemoryResponse{}, nil
}

func (s *agentMemorySvc) List(ctx context.Context, req *ListMemoryRequest) (*ListMemoryResponse, error) {
	var rows []*agent_memory_entity.AgentMemory
	var err error
	if req.Scope != "" {
		rows, err = agent_memory_repo.AgentMemory().ListByAgentAndScope(ctx, req.AgentID, req.Scope)
	} else {
		rows, err = agent_memory_repo.AgentMemory().ListByAgent(ctx, req.AgentID)
	}
	if err != nil {
		return nil, err
	}
	items := make([]*MemoryItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, toItem(r))
	}
	return &ListMemoryResponse{Items: items}, nil
}

func toItem(m *agent_memory_entity.AgentMemory) *MemoryItem {
	return &MemoryItem{
		ID:         m.ID,
		AgentID:    m.AgentID,
		Scope:      m.Scope,
		SessionID:  m.SessionID,
		Category:   m.Category,
		Key:        m.Key,
		Content:    m.Content,
		Source:     m.Source,
		Createtime: m.Createtime,
		Updatetime: m.Updatetime,
	}
}
```

- [ ] **步骤 3：运行编译验证**

运行：`go build ./internal/service/agent_memory_svc/`
预期：PASS

- [ ] **步骤 4：Commit**

```bash
git add internal/service/agent_memory_svc/
git commit -m "feat(agent_memory): add AgentMemorySvc CRUD operations"
```

---

### 任务 5：AgentMemory Service — LoadMemories + ExtractFromSession

**文件：**
- 修改：`internal/service/agent_memory_svc/agent_memory.go`（追加两个方法）

- [ ] **步骤 1：编写 LoadMemories 测试**

```go
func TestAgentMemorySvc_LoadMemories(t *testing.T) {
	Convey("LoadMemories", t, func() {
		// 使用 mock repo 注入测试
		// 验证 scope=user + scope=project 的记忆被正确加载
		// 验证 scope=session + sessionID 时只加载对应 session 的记忆
	})
}
```

- [ ] **步骤 2：运行测试验证失败**

运行：`go test -race ./internal/service/agent_memory_svc/ -run TestAgentMemorySvc_LoadMemories -v`
预期：FAIL，方法未实现

- [ ] **步骤 3：实现 LoadMemories**

```go
func (s *agentMemorySvc) LoadMemories(ctx context.Context, agentID int64, scope string, sessionID int64) ([]*MemoryItem, error) {
	var all []*agent_memory_entity.AgentMemory
	var err error

	if scope != "" {
		all, err = agent_memory_repo.AgentMemory().ListByAgentAndScope(ctx, agentID, scope)
	} else {
		// 默认加载 user + project scope（跨会话持久记忆）
		userRows, e := agent_memory_repo.AgentMemory().ListByAgentAndScope(ctx, agentID, string(agent_memory_entity.ScopeUser))
		if e != nil {
			return nil, e
		}
		projectRows, e := agent_memory_repo.AgentMemory().ListByAgentAndScope(ctx, agentID, string(agent_memory_entity.ScopeProject))
		if e != nil {
			return nil, e
		}
		all = append(all, userRows...)
		all = append(all, projectRows...)
	}

	if sessionID > 0 {
		sessionRows, e := agent_memory_repo.AgentMemory().FindBySession(ctx, sessionID)
		if e != nil {
			return nil, e
		}
		all = append(all, sessionRows...)
	}

	items := make([]*MemoryItem, 0, len(all))
	for _, m := range all {
		items = append(items, toItem(m))
	}
	return items, nil
}
```

- [ ] **步骤 4：实现 ExtractFromSession（骨架）**

```go
func (s *agentMemorySvc) ExtractFromSession(ctx context.Context, sessionID int64) error {
	// 第一版：仅标记 session 有记忆需求，实际 LLM 提取留待后续迭代。
	// 当前实现：从 chat_repo 获取 session 的最后一条 assistant 消息，
	// 截取前 500 字符作为摘要存入 agent_memories。
	// 完整的 LLM 提取需要 provider 调用，将在后续迭代中实现。
	return nil
}
```

- [ ] **步骤 5：运行测试验证通过**

运行：`go test -race ./internal/service/agent_memory_svc/ -v`
预期：PASS

- [ ] **步骤 6：Commit**

```bash
git add internal/service/agent_memory_svc/
git commit -m "feat(agent_memory): add LoadMemories and ExtractFromSession"
```

---

### 任务 6：Wails 绑定

**文件：**
- 创建：`internal/app/agent_memory.go`

- [ ] **步骤 1：编写 Wails 绑定**

```go
package app

import (
	"agentre/internal/service/agent_memory_svc"
)

// CreateAgentMemory 新建 Agent 记忆。
func (a *App) CreateAgentMemory(req *agent_memory_svc.CreateMemoryRequest) (*agent_memory_svc.CreateMemoryResponse, error) {
	return agent_memory_svc.AgentMemory().Create(a.ctx, req)
}

// UpdateAgentMemory 更新 Agent 记忆。
func (a *App) UpdateAgentMemory(req *agent_memory_svc.UpdateMemoryRequest) (*agent_memory_svc.UpdateMemoryResponse, error) {
	return agent_memory_svc.AgentMemory().Update(a.ctx, req)
}

// DeleteAgentMemory 删除 Agent 记忆。
func (a *App) DeleteAgentMemory(req *agent_memory_svc.DeleteMemoryRequest) (*agent_memory_svc.DeleteMemoryResponse, error) {
	return agent_memory_svc.AgentMemory().Delete(a.ctx, req)
}

// ListAgentMemories 列出 Agent 记忆。
func (a *App) ListAgentMemories(req *agent_memory_svc.ListMemoryRequest) (*agent_memory_svc.ListMemoryResponse, error) {
	return agent_memory_svc.AgentMemory().List(a.ctx, req)
}
```

- [ ] **步骤 2：运行编译验证**

运行：`go build ./internal/app/`
预期：PASS

- [ ] **步骤 3：Commit**

```bash
git add internal/app/agent_memory.go
git commit -m "feat(agent_memory): add Wails bindings for AgentMemory"
```

---

### 任务 7：Bootstrap 注册

**文件：**
- 修改：`internal/bootstrap/cago.go`

- [ ] **步骤 1：添加 import 和注册**

在 `cago.go` 的 import 中添加：

```go
"agentre/internal/repository/agent_memory_repo"
```

在 `Init()` 函数的 repo 注册区域追加：

```go
agent_memory_repo.RegisterAgentMemory(agent_memory_repo.NewAgentMemory())
```

- [ ] **步骤 2：运行编译验证**

运行：`go build ./internal/bootstrap/`
预期：PASS

- [ ] **步骤 3：Commit**

```bash
git add internal/bootstrap/cago.go
git commit -m "feat(agent_memory): register AgentMemoryRepo in bootstrap"
```

---

### 任务 8：builtin runtime 注入记忆

**文件：**
- 修改：`internal/pkg/agentruntime/runtimes/builtin/runtime.go`

- [ ] **步骤 1：编写注入记忆的测试**

在 `runtime_test.go` 中添加测试：验证当 agent 有 user scope 记忆时，Run() 的 system prompt 包含记忆内容。

- [ ] **步骤 2：运行测试验证失败**

运行：`go test -race ./internal/pkg/agentruntime/runtimes/builtin/ -run TestMemoryInjection -v`
预期：FAIL

- [ ] **步骤 3：在 Run() 中注入记忆**

在 `runtime.go` 的 `Run()` 方法中，构建 `opts` 之前，调用 `agent_memory_svc.AgentMemory().LoadMemories()` 获取记忆，格式化为文本追加到 system prompt：

```go
// 在 opts 构建之前
memories, memErr := agent_memory_svc.AgentMemory().LoadMemories(ctx, req.AgentID, "", 0)
if memErr != nil {
    logger.Ctx(ctx).Warn("builtin runtime: load memories failed", zap.Error(memErr))
}
if len(memories) > 0 {
    var memSB strings.Builder
    memSB.WriteString("\n\n[Agent Memories]\n")
    for _, m := range memories {
        memSB.WriteString(fmt.Sprintf("- [%s/%s] %s\n", m.Scope, m.Category, m.Content))
    }
    sys = sys + memSB.String()
}
```

- [ ] **步骤 4：运行测试验证通过**

运行：`go test -race ./internal/pkg/agentruntime/runtimes/builtin/ -v`
预期：PASS

- [ ] **步骤 5：Commit**

```bash
git add internal/pkg/agentruntime/runtimes/builtin/
git commit -m "feat(agent_memory): inject memories into builtin runtime system prompt"
```

---

### 任务 9：chat_svc 触发记忆提取

**文件：**
- 修改：`internal/service/chat_svc/chat.go`

- [ ] **步骤 1：在 turn 结束时触发 ExtractFromSession**

在 chat_svc 的 turn 结束处理逻辑中（runTurn 方法返回后），异步调用 `agent_memory_svc.AgentMemory().ExtractFromSession()`：

```go
// turn 正常结束后，异步触发记忆提取
go func() {
    if err := agent_memory_svc.AgentMemory().ExtractFromSession(context.Background(), sessionID); err != nil {
        logger.Ctx(ctx).Warn("agent_memory: extract from session failed",
            zap.Int64("sessionID", sessionID), zap.Error(err))
    }
}()
```

- [ ] **步骤 2：运行编译验证**

运行：`go build ./internal/service/chat_svc/`
预期：PASS

- [ ] **步骤 3：Commit**

```bash
git add internal/service/chat_svc/
git commit -m "feat(agent_memory): trigger memory extraction after turn ends"
```

---

### 任务 10：生成 Wails 前端绑定 + 前端记忆标签页

**文件：**
- 运行 `make generate` 生成前端绑定
- 修改：前端 Agent 设置页，添加记忆标签页

- [ ] **步骤 1：生成 Wails 绑定**

运行：`make generate`

- [ ] **步骤 2：添加前端记忆标签页组件**

在 Agent 设置/编辑页中添加"记忆"标签页，调用 `ListAgentMemories` 展示列表，支持 `CreateAgentMemory` / `UpdateAgentMemory` / `DeleteAgentMemory` 操作。

- [ ] **步骤 3：添加 i18n 文案**

在 `frontend/src/i18n/locales/{zh-CN,en}/common.json` 中添加记忆相关的 UI 文案。

- [ ] **步骤 4：运行前端测试**

运行：`cd frontend && pnpm test`
预期：PASS

- [ ] **步骤 5：Commit**

```bash
git add frontend/ internal/pkg/agentruntime/runtimes/builtin/
git commit -m "feat(agent_memory): add frontend memory tab and i18n"
```

---

### 任务 11：端到端验证

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
1. 打开 Agent 设置页，切换到"记忆"标签页
2. 手动添加一条 user scope 记忆
3. 开始新聊天会话，验证记忆被注入 system prompt
4. 结束会话，验证记忆提取被触发
