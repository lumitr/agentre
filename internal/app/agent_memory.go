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
