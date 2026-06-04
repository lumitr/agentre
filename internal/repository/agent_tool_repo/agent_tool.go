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
