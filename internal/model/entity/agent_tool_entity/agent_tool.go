// Package agent_tool_entity 维护 Agent 工具的充血实体。
package agent_tool_entity

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/cago-frame/cago/pkg/consts"
	"github.com/cago-frame/cago/pkg/i18n"

	"agentre/internal/pkg/code"
)

// ExecutorType 工具执行器类型。
type ExecutorType string

const (
	ExecutorBuiltin ExecutorType = "builtin"
	ExecutorHTTP    ExecutorType = "http"
	ExecutorMCP     ExecutorType = "mcp"
)

var validExecutorTypes = map[ExecutorType]bool{
	ExecutorBuiltin: true,
	ExecutorHTTP:    true,
	ExecutorMCP:     true,
}

// AgentTool 一条 Agent 工具配置记录。
type AgentTool struct {
	ID           int64  `gorm:"column:id;primaryKey;autoIncrement"`
	AgentID      int64  `gorm:"column:agent_id;type:bigint;not null;default:0"`
	Name         string `gorm:"column:name;type:text;not null"`
	Description  string `gorm:"column:description;type:text;not null;default:''"`
	ParamSchema  string `gorm:"column:param_schema;type:text;not null;default:'{}'"`
	ExecutorType string `gorm:"column:executor_type;type:text;not null"`
	ExecutorConf string `gorm:"column:executor_conf;type:text;not null;default:'{}'"`
	Enabled      bool   `gorm:"column:enabled;type:boolean;not null;default:true"`
	SortOrder    int    `gorm:"column:sort_order;type:int;not null;default:0"`
	Status       int    `gorm:"column:status;type:int;not null;default:1"`
	Createtime   int64  `gorm:"column:createtime;type:bigint;not null;default:0"`
	Updatetime   int64  `gorm:"column:updatetime;type:bigint;not null;default:0"`
}

func (*AgentTool) TableName() string { return "agent_tools" }

func (a *AgentTool) IsActive() bool { return a != nil && a.Status == consts.ACTIVE }

// Check 字段校验。
func (a *AgentTool) Check(ctx context.Context) error {
	if a == nil {
		return i18n.NewError(ctx, code.AgentToolNotFound)
	}
	if strings.TrimSpace(a.Name) == "" {
		return i18n.NewError(ctx, code.InvalidParameter)
	}
	if !validExecutorTypes[ExecutorType(a.ExecutorType)] {
		return i18n.NewError(ctx, code.AgentToolInvalidExecutorType)
	}
	if !isValidJSON(a.ParamSchema) {
		return i18n.NewError(ctx, code.AgentToolInvalidParamSchema)
	}
	if a.ExecutorConf != "" && a.ExecutorConf != "{}" && !isValidJSON(a.ExecutorConf) {
		return i18n.NewError(ctx, code.AgentToolInvalidExecutorConf)
	}
	return nil
}

func isValidJSON(s string) bool {
	var v json.RawMessage
	return json.Unmarshal([]byte(s), &v) == nil
}
