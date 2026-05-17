package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool"
	mcpsdk "github.com/mark3labs/mcp-go/mcp"
)

func newTestManager(servers ...*Server) *Manager {
	return &Manager{
		servers: servers,
		byName:  make(map[string]*adapterTool),
	}
}

func stubServer(name string, tools []mcpsdk.Tool) *Server {
	return &Server{
		Name: name,
		testListTools: func(context.Context) ([]mcpsdk.Tool, error) {
			return tools, nil
		},
		testClose: func() error { return nil },
	}
}

func TestNewManagerFromConfig_empty(t *testing.T) {
	m, err := NewManagerFromConfig(context.Background(), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if m.ToolCount() != 0 {
		t.Fatal("expected no tools")
	}
}

func TestNewManager_empty(t *testing.T) {
	m, err := NewManager(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.ToolCount() != 0 {
		t.Fatalf("tool count = %d", m.ToolCount())
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestNewManager_invalidServerName(t *testing.T) {
	_, err := NewManager(context.Background(), []config.MCPServerConfig{
		{Name: "Bad-Name", Command: "true"},
	})
	if err == nil {
		t.Fatal("expected invalid server name error")
	}
}

func TestNewManager_duplicateServerName(t *testing.T) {
	_, err := NewManager(context.Background(), []config.MCPServerConfig{
		{Name: "fs", Command: "true"},
		{Name: "fs", Command: "true"},
	})
	if err == nil {
		t.Fatal("expected duplicate server name error")
	}
}

func TestNewManager_missingCommand(t *testing.T) {
	_, err := NewManager(context.Background(), []config.MCPServerConfig{
		{Name: "fs"},
	})
	if err == nil {
		t.Fatal("expected missing command error")
	}
}

func TestManager_DiscoverTools_registersTools(t *testing.T) {
	srv := stubServer("fs", []mcpsdk.Tool{
		{Name: "read_file", Description: "read"},
		{Name: "write_file", Description: "write"},
	})
	m := newTestManager(srv)
	if err := m.DiscoverTools(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if m.ToolCount() != 2 {
		t.Fatalf("tool count = %d", m.ToolCount())
	}
	if got := m.ServerNames(); len(got) != 1 || got[0] != "fs" {
		t.Fatalf("server names = %v", got)
	}
}

func TestManager_DiscoverTools_duplicateToolName(t *testing.T) {
	srv := stubServer("fs", []mcpsdk.Tool{
		{Name: "read_file"},
		{Name: "read_file"},
	})
	m := newTestManager(srv)
	err := m.DiscoverTools(context.Background(), false)
	if err == nil {
		t.Fatal("expected duplicate tool name error")
	}
}

func TestManager_DiscoverTools_listError(t *testing.T) {
	srv := &Server{
		Name: "fs",
		testListTools: func(context.Context) ([]mcpsdk.Tool, error) {
			return nil, errors.New("list failed")
		},
	}
	m := newTestManager(srv)
	err := m.DiscoverTools(context.Background(), false)
	if err == nil {
		t.Fatal("expected list tools error")
	}
}

func TestManager_Register(t *testing.T) {
	srv := stubServer("fs", []mcpsdk.Tool{{Name: "ping"}})
	m := newTestManager(srv)
	if err := m.DiscoverTools(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	reg := tool.NewRegistry()
	m.Register(reg)
	if _, ok := reg.Get(ToolName("fs", "ping")); !ok {
		t.Fatal("expected MCP tool registered")
	}
}

func TestManager_IsWriteTool_fromAdapterLevel(t *testing.T) {
	destructive := true
	srv := stubServer("fs", []mcpsdk.Tool{{
		Name: "delete_item",
		Annotations: mcpsdk.ToolAnnotation{
			DestructiveHint: &destructive,
		},
	}})
	m := newTestManager(srv)
	if err := m.DiscoverTools(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	name := ToolName("fs", "delete_item")
	if !m.IsWriteTool(name) {
		t.Fatal("destructive MCP tool should require write permission")
	}
	readName := ToolName("fs", "read_file")
	if m.IsWriteTool(readName) {
		t.Fatal("unknown tool should fall back to name heuristics")
	}
}

func TestManager_IsWriteTool_fallbackHeuristic(t *testing.T) {
	m := newTestManager()
	if !m.IsWriteTool(ToolName("fs", "write_file")) {
		t.Fatal("write_file heuristic should match without discovery")
	}
}

func TestManager_Close_clearsState(t *testing.T) {
	closed := false
	srv := &Server{
		Name: "fs",
		testListTools: func(context.Context) ([]mcpsdk.Tool, error) {
			return nil, nil
		},
		testClose: func() error {
			closed = true
			return nil
		},
	}
	m := newTestManager(srv)
	if err := m.DiscoverTools(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if !closed {
		t.Fatal("expected server Close to run")
	}
	if m.ToolCount() != 0 {
		t.Fatal("tools should be cleared after Close")
	}
}

func TestAdapterTool_descriptionFallback(t *testing.T) {
	srv := stubServer("fs", nil)
	ad := newAdapterTool(srv, mcpsdk.Tool{Name: "ping"}, false)
	if ad.Description() == "" {
		t.Fatal("expected fallback description")
	}
	if !strings.Contains(ad.Description(), "server fs") {
		t.Fatalf("desc = %q", ad.Description())
	}
}

func TestAdapterTool_execute(t *testing.T) {
	srv := &Server{
		Name: "fs",
		testCallTool: func(_ context.Context, tool string, args json.RawMessage) (string, error) {
			if tool != "echo" {
				t.Fatalf("tool = %q", tool)
			}
			return `{"ok":true,"args":` + string(args) + `}`, nil
		},
	}
	ad := newAdapterTool(srv, mcpsdk.Tool{Name: "echo", Description: "echo back"}, false)
	out, err := ad.Execute(context.Background(), json.RawMessage(`{"x":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if out == "" {
		t.Fatal("expected non-empty result")
	}
	if ad.PermissionLevel() != permission.LevelLow {
		t.Fatalf("level = %v", ad.PermissionLevel())
	}
}
