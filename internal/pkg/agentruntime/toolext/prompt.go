package toolext

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FormatToolsPrompt 将工具定义格式化为 system prompt 片段。
func FormatToolsPrompt(defs []ToolDef) string {
	if len(defs) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n\n[Available Tools]\n")
	sb.WriteString("You have access to the following custom tools. Use them when appropriate:\n\n")
	for _, d := range defs {
		sb.WriteString(fmt.Sprintf("### %s\n%s\n\nParameters:\n```json\n%s\n```\n\n", d.Name, d.Description, indentJSON(d.ParamSchema)))
	}
	sb.WriteString("To use a tool, include a tool_use block with the tool name and parameters in your response.\n")
	return sb.String()
}

func indentJSON(raw json.RawMessage) string {
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return string(raw)
	}
	return string(pretty)
}
