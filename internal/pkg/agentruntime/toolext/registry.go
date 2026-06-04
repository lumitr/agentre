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

func RegisterBuiltin(name string, exec ToolExecutor) {
	mu.Lock()
	defer mu.Unlock()
	builtins[name] = exec
}

func LookupBuiltin(name string) (ToolExecutor, bool) {
	mu.RLock()
	defer mu.RUnlock()
	e, ok := builtins[name]
	return e, ok
}

// LoadTools 加载指定 agent 启用的工具定义（全局 + agent 专属）。
func LoadTools(ctx context.Context, agentID int64) ([]ToolDef, error) {
	var defs []ToolDef

	global, err := agent_tool_repo.AgentTool().ListEnabledGlobal(ctx)
	if err != nil {
		return nil, fmt.Errorf("toolext: load global tools: %w", err)
	}
	for _, t := range global {
		defs = append(defs, toolToDef(t))
	}

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
