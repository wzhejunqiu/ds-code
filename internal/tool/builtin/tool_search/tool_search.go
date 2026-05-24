// Package tool_search provides a built-in tool that lets the LLM look up
// full schemas for deferred tools by name.
package tool_search

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
)

// ToolSearchTool returns the full schema of a deferred tool by name.
type ToolSearchTool struct {
	Registry *tool.Registry
	Strict   bool
}

func (t *ToolSearchTool) Name() string        { return "tool_search" }
func (t *ToolSearchTool) IsReadOnly() bool    { return true }
func (t *ToolSearchTool) IsConcurrencySafe() bool { return true }

func (t *ToolSearchTool) Description() string {
	return "按名称查找工具的完整定义（仅用于延迟加载的工具）"
}

func (t *ToolSearchTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"tool_name": map[string]any{"type": "string", "description": "要查找的工具名称"},
	}, []string{"tool_name"}, t.Strict)
}

func (t *ToolSearchTool) PermissionLevel() permission.Level { return permission.LevelLow }

func (t *ToolSearchTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		ToolName string `json:"tool_name"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	if in.ToolName == "" {
		return "", fmt.Errorf("tool_name is required")
	}
	tl, ok := t.Registry.Get(in.ToolName)
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", in.ToolName)
	}
	schema, _ := json.MarshalIndent(tl.Schema(), "", "  ")
	return fmt.Sprintf("Tool: %s\nDescription: %s\nSchema:\n%s", tl.Name(), tl.Description(), string(schema)), nil
}
