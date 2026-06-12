package tool

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/permission"
)

type stubTool struct {
	name string
}

func (s *stubTool) Name() string        { return s.name }
func (s *stubTool) Description() string { return "desc" }
func (s *stubTool) Schema() map[string]any {
	return ObjectSchema(map[string]any{"full": map[string]any{"type": "string"}}, []string{"full"}, false)
}
func (s *stubTool) Execute(ctx context.Context, _ json.RawMessage) (string, error) { return "ok", nil }
func (s *stubTool) PermissionLevel() permission.Level                              { return permission.LevelLow }

func TestRegistry_DeferredDefinitions(t *testing.T) {
	reg := NewRegistry()
	reg.Register(WrapDeferred(&stubTool{name: "mcp__srv__tool"}))
	reg.Register(&stubTool{name: "read_file"})

	for _, d := range reg.Definitions() {
		props, _ := d.Parameters["properties"].(map[string]any)
		switch d.Name {
		case "mcp__srv__tool":
			if props["full"] != nil {
				t.Fatal("deferred tool should use stub schema")
			}
		case "read_file":
			if props["full"] == nil {
				t.Fatal("read_file should expose full schema")
			}
		}
	}
	full, ok := reg.FullSchema("mcp__srv__tool")
	if !ok || full["properties"] == nil {
		t.Fatal("expected full schema via FullSchema")
	}
	if !reg.HasDeferredTools() {
		t.Fatal("expected HasDeferredTools true")
	}
}
