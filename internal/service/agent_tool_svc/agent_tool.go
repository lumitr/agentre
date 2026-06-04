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
