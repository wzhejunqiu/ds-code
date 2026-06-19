package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	mcpsdk "github.com/mark3labs/mcp-go/mcp"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
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
	m, err := NewManagerFromConfig(context.Background(), nil, false, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.ToolCount() != 0 {
		t.Fatal("expected no tools")
	}
}

func TestNewManager_empty(t *testing.T) {
	m, err := NewManager(context.Background(), nil, nil)
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
	}, nil)
	if err == nil {
		t.Fatal("expected invalid server name error")
	}
}

func TestNewManager_duplicateServerName(t *testing.T) {
	_, err := NewManager(context.Background(), []config.MCPServerConfig{
		{Name: "fs", Command: "true"},
		{Name: "fs", Command: "true"},
	}, nil)
	if err == nil {
		t.Fatal("expected duplicate server name error")
	}
}

func TestManager_DiscoverTools_duplicateServerNameStillFails(t *testing.T) {
	TestNewManager_duplicateServerName(t)
}

func TestNewManager_missingCommand(t *testing.T) {
	_, err := NewManager(context.Background(), []config.MCPServerConfig{
		{Name: "fs"},
	}, nil)
	if err == nil {
		t.Fatal("expected missing command error")
	}
}

func TestManager_DiscoverTools_bareNames(t *testing.T) {
	srv := stubServer("fs", []mcpsdk.Tool{
		{Name: "semantic_search_nodes", Description: "search"},
		{Name: "ping", Description: "ping"},
	})
	m := newTestManager(srv)
	if err := m.DiscoverTools(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if m.ToolCount() != 2 {
		t.Fatalf("tool count = %d", m.ToolCount())
	}
	if _, ok := m.byName["semantic_search_nodes"]; !ok {
		t.Fatal("expected bare name in byName")
	}
	reg := tool.NewRegistry()
	m.Register(reg)
	if _, ok := reg.Get("semantic_search_nodes"); !ok {
		t.Fatal("expected bare name registered")
	}
	if _, ok := reg.Get(ToolName("fs", "semantic_search_nodes")); ok {
		t.Fatal("legacy normalized name must not be registered")
	}
}

func TestManager_DiscoverTools_inServerDuplicateSkipped(t *testing.T) {
	srv := stubServer("fs", []mcpsdk.Tool{
		{Name: "read_dup"},
		{Name: "read_dup"},
	})
	m := newTestManager(srv)
	if err := m.DiscoverTools(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if m.ToolCount() != 1 {
		t.Fatalf("tool count = %d, want 1", m.ToolCount())
	}
	skipped := m.SkippedTools()
	if len(skipped) != 1 || skipped[0].Reason != SkipInServerDuplicate {
		t.Fatalf("skipped = %+v", skipped)
	}
}

func TestManager_DiscoverTools_crossServerDuplicateBothSkipped(t *testing.T) {
	a := stubServer("a", []mcpsdk.Tool{{Name: "search"}})
	b := stubServer("b", []mcpsdk.Tool{{Name: "search"}})
	m := newTestManager(a, b)
	if err := m.DiscoverTools(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if m.ToolCount() != 0 {
		t.Fatalf("tool count = %d", m.ToolCount())
	}
	skipped := m.SkippedTools()
	if len(skipped) != 2 {
		t.Fatalf("skipped count = %d", len(skipped))
	}
	for _, s := range skipped {
		if s.Reason != SkipCrossServerDuplicate || s.Tool != "search" {
			t.Fatalf("unexpected skip %+v", s)
		}
	}
	reg := tool.NewRegistry()
	m.Register(reg)
	if reg.IsMCPTool("search") {
		t.Fatal("cross-server duplicate must not register")
	}
}

func TestManager_DiscoverTools_builtinConflictSkipped(t *testing.T) {
	srv := stubServer("fs", []mcpsdk.Tool{{Name: "grep", Description: "mcp grep"}})
	m := newTestManager(srv)
	if err := m.DiscoverTools(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	reg := tool.NewRegistry()
	reg.Register(&stubBuiltinTool{name: "grep"})
	m.Register(reg)
	if m.RegisteredToolCount() != 0 {
		t.Fatalf("registered = %d", m.RegisteredToolCount())
	}
	skipped := m.SkippedTools()
	if len(skipped) != 1 || skipped[0].Reason != SkipBuiltinConflict {
		t.Fatalf("skipped = %+v", skipped)
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
	if _, ok := reg.Get("ping"); !ok {
		t.Fatal("expected MCP tool registered by bare name")
	}
	server, ok := reg.MCPServerForTool("ping")
	if !ok || server != "fs" {
		t.Fatalf("server = %q ok=%v", server, ok)
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
	reg := tool.NewRegistry()
	m.Register(reg)
	if !m.IsWriteTool("delete_item") {
		t.Fatal("destructive MCP tool should require write permission")
	}
}

func TestManager_IsWriteTool_builtinConflictNotWrite(t *testing.T) {
	destructive := true
	srv := stubServer("fs", []mcpsdk.Tool{{
		Name: "grep",
		Annotations: mcpsdk.ToolAnnotation{
			DestructiveHint: &destructive,
		},
	}})
	m := newTestManager(srv)
	if err := m.DiscoverTools(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	reg := tool.NewRegistry()
	reg.Register(&stubBuiltinTool{name: "grep"})
	m.Register(reg)
	if m.IsWriteTool("grep") {
		t.Fatal("builtin-conflicted MCP grep must not classify builtin grep as write")
	}
	if m.RegisteredToolCount() != 0 {
		t.Fatalf("registered = %d", m.RegisteredToolCount())
	}
}

func TestManager_IsWriteTool_builtinNotMisclassified(t *testing.T) {
	srv := stubServer("fs", []mcpsdk.Tool{{Name: "read_file"}})
	m := newTestManager(srv)
	if err := m.DiscoverTools(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	reg := tool.NewRegistry()
	m.Register(reg)
	if m.IsWriteTool("grep") {
		t.Fatal("builtin grep must not be classified via MCP manager")
	}
	if m.IsWriteTool("read_file") {
		t.Fatal("read_file MCP adapter should not be write")
	}
	if m.IsWriteTool(ToolName("fs", "write_file")) {
		t.Fatal("legacy normalized name must not trigger write without adapter")
	}
}

func TestManager_IsWriteTool_fallbackHeuristicRemoved(t *testing.T) {
	m := newTestManager()
	if m.IsWriteTool("write_file") {
		t.Fatal("undiscovered write_file must not match global heuristic")
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
	ad := newAdapterTool(srv, mcpsdk.Tool{Name: "ping"}, false, false)
	if ad.Description() == "" {
		t.Fatal("expected fallback description")
	}
	if !strings.Contains(ad.Description(), "server fs") {
		t.Fatalf("desc = %q", ad.Description())
	}
	if ad.Name() != "ping" {
		t.Fatalf("name = %q", ad.Name())
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
	ad := newAdapterTool(srv, mcpsdk.Tool{Name: "echo", Description: "echo back"}, false, false)
	out, err := ad.Execute(context.Background(), json.RawMessage(`{"x":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if out == "" {
		t.Fatal("expected non-empty result")
	}
	if ad.PermissionLevel() != permission.LevelMedium {
		t.Fatalf("level = %v", ad.PermissionLevel())
	}
}

type stubBuiltinTool struct {
	name string
}

func (s *stubBuiltinTool) Name() string        { return s.name }
func (s *stubBuiltinTool) Description() string { return "builtin" }
func (s *stubBuiltinTool) Schema() map[string]any {
	return tool.ObjectSchema(nil, nil, false)
}
func (s *stubBuiltinTool) Execute(context.Context, json.RawMessage) (string, error) {
	return "", nil
}
func (s *stubBuiltinTool) PermissionLevel() permission.Level { return permission.LevelLow }
