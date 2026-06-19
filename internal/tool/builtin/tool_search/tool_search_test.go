package tool_search

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
)

type stubTool struct{ name string }

func (s stubTool) Name() string        { return s.name }
func (s stubTool) Description() string { return "mcp tool" }
func (s stubTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{"arg": map[string]any{"type": "string"}}, []string{"arg"}, false)
}
func (s stubTool) Execute(context.Context, json.RawMessage) (string, error) { return "", nil }
func (s stubTool) PermissionLevel() permission.Level                        { return permission.LevelLow }

func TestToolSearch_returnsFullSchema(t *testing.T) {
	reg := tool.NewRegistry()
	reg.Register(tool.WrapDeferred(stubTool{name: "mcp__x__y"}))
	ts := &ToolSearchTool{Registry: reg}
	out, err := ts.Execute(context.Background(), json.RawMessage(`{"tool_name":"mcp__x__y"}`))
	if err != nil {
		t.Fatal(err)
	}
	if out == "" || !strings.Contains(out, `"arg"`) {
		t.Fatalf("expected full schema in output, got %q", out)
	}
}

func TestToolSearch_bareMCPName(t *testing.T) {
	reg := tool.NewRegistry()
	reg.RegisterMCPTool(tool.WrapDeferred(stubTool{name: "semantic_search_nodes"}), "graph")
	ts := &ToolSearchTool{Registry: reg}
	out, err := ts.Execute(context.Background(), json.RawMessage(`{"tool_name":"semantic_search_nodes"}`))
	if err != nil {
		t.Fatal(err)
	}
	if out == "" || !strings.Contains(out, `"arg"`) {
		t.Fatalf("expected full schema in output, got %q", out)
	}
	if strings.Contains(out, "unknown tool") {
		t.Fatalf("unexpected unknown tool: %q", out)
	}
}
