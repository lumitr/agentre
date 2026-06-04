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
		AgentID:    req.AgentID,
		Scope:      strings.TrimSpace(req.Scope),
		SessionID:  req.SessionID,
		Category:   strings.TrimSpace(req.Category),
		Key:        strings.TrimSpace(req.Key),
		Content:    strings.TrimSpace(req.Content),
		Source:     string(agent_memory_entity.SourceManual),
		Status:     1,
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

func (s *agentMemorySvc) LoadMemories(_ context.Context, _ int64, _ string, _ int64) ([]*MemoryItem, error) {
	// TODO: 任务 5 实现
	return nil, nil
}

func (s *agentMemorySvc) ExtractFromSession(_ context.Context, _ int64) error {
	// TODO: 任务 5 实现
	return nil
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
