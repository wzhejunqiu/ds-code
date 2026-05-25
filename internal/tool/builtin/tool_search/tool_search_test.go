package tool_search

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
)

type stubTool struct{}

func (stubTool) Name() string        { return "mcp__x__y" }
func (stubTool) Description() string { return "mcp tool" }
func (stubTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{"arg": map[string]any{"type": "string"}}, []string{"arg"}, false)
}
func (stubTool) Execute(context.Context, json.RawMessage) (string, error) { return "", nil }
func (stubTool) PermissionLevel() permission.Level                        { return permission.LevelLow }

func TestToolSearch_returnsFullSchema(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Register(tool.WrapDeferred(stubTool{}))
	ts := &ToolSearchTool{Registry: reg}
	out, err := ts.Execute(context.Background(), json.RawMessage(`{"tool_name":"mcp__x__y"}`))
	if err != nil {
		t.Fatal(err)
	}
	if out == "" || !strings.Contains(out, `"arg"`) {
		t.Fatalf("expected full schema in output, got %q", out)
	}
}
