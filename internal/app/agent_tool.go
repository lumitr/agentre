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
