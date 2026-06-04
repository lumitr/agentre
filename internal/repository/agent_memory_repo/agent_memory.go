// Package agent_memory_repo 提供 AgentMemory 的持久化访问。
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
		Where("session_id = ? AND status = ?", sessionID, consts.ACTIVE).
		Order("category ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *agentMemoryRepo) UpsertByAgentAndKey(ctx context.Context, m *agent_memory_entity.AgentMemory) error {
	existing, err := r.FindByAgentAndKey(ctx, m.AgentID, m.Scope, m.Key)
	if err != nil {
		return err
	}
	if existing != nil {
		m.ID = existing.ID
		return r.Update(ctx, m)
	}
	return r.Create(ctx, m)
}
