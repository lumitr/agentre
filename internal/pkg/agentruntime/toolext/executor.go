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
