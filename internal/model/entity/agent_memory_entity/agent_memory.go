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
