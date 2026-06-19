package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/datadir"
	"github.com/wzhejunqiu/ds-code/internal/llm"
	"github.com/wzhejunqiu/ds-code/internal/mcp/resultstore"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/testutil"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	toolresultpkg "github.com/wzhejunqiu/ds-code/internal/toolresult"
)

type stubMCPTool struct {
	name string
	out  string
}

func (s *stubMCPTool) Name() string        { return s.name }
func (s *stubMCPTool) Description() string { return "stub mcp" }
func (s *stubMCPTool) Schema() map[string]any {
	return map[string]any{"type": "object"}
}
func (s *stubMCPTool) Execute(context.Context, json.RawMessage) (string, error) {
	return s.out, nil
}
func (s *stubMCPTool) PermissionLevel() permission.Level { return permission.LevelLow }

func mcpRunner(t *testing.T, root string, maxChars int) (*Runner, *tool.Registry) {
	t.Helper()
	testutil.IsolatedHome(t)
	reg := tool.NewRegistry()
	reg.RegisterMCPTool(&stubMCPTool{name: "mcp_tool", out: "unused"}, "test-server")
	cfg := &config.Config{
		ProjectRoot: root,
		Context:     config.ContextConfig{ToolResultMaxChars: maxChars},
	}
	return &Runner{
		Tools:      reg,
		MCPResults: &resultstore.Store{ProjectRoot: root},
		Cfg:        cfg,
		Perm:       permission.NewEngine("auto", root, false),
	}, reg
}

func formattedBody(body string) string {
	return toolresultpkg.FormatToolResult("mcp_tool", "call_1", body)
}

func TestFinalizeToolResult_mcpUnderLimit(t *testing.T) {
	root := t.TempDir()
	r, _ := mcpRunner(t, root, 100_000)
	body := strings.Repeat("x", 50_000)
	tc := llm.ToolCall{ID: "call_1", Name: "mcp_tool"}

	got := r.finalizeToolResult("sess-1", tc, formattedBody(body))
	if got != formattedBody(body) {
		t.Fatal("under limit should return full formatted body without hint")
	}
	dir, err := datadir.MCPResultSessionDir(root, "sess-1")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected spill file, got %d entries", len(entries))
	}
}

func TestFinalizeToolResult_mcpSpillAndTruncate(t *testing.T) {
	root := t.TempDir()
	r, _ := mcpRunner(t, root, 100_000)
	inner := strings.Repeat("y", 150_000)
	tc := llm.ToolCall{ID: "call_abc", Name: "mcp_tool"}

	got := r.finalizeToolResult("sess-1", tc, formattedBody(inner))
	if len(got) > 100_000 {
		t.Fatalf("session content len %d exceeds max 100000", len(got))
	}
	if !strings.Contains(got, "read_file") {
		t.Fatal("truncated result should contain spill hint")
	}
}

func TestFinalizeToolResult_mcpSpillHintBudget(t *testing.T) {
	root := t.TempDir()
	r, _ := mcpRunner(t, root, 500)
	inner := strings.Repeat("z", 10_000)
	tc := llm.ToolCall{ID: "call_1", Name: "mcp_tool"}

	got := r.finalizeToolResult("sess-1", tc, formattedBody(inner))
	if len(got) > 500 {
		t.Fatalf("len %d > max 500", len(got))
	}
}

func TestFinalizeToolResult_mcpHintBudgetLongPath(t *testing.T) {
	root := t.TempDir()
	r, _ := mcpRunner(t, root, 100_000)
	inner := strings.Repeat("a", 150_000)
	tc := llm.ToolCall{ID: "call_long", Name: "mcp_tool"}

	got := r.finalizeToolResult("sess-1", tc, formattedBody(inner))
	if len(got) > 100_000 {
		t.Fatalf("len %d > max 100000", len(got))
	}
	if !strings.Contains(got, "mcp-result") {
		t.Fatalf("hint should contain spill path: %q", got)
	}
}

func TestFinalizeToolResult_builtinNoSpill(t *testing.T) {
	root := t.TempDir()
	testutil.IsolatedHome(t)
	reg := tool.NewRegistry()
	reg.Register(&stubMCPTool{name: "read_file", out: strings.Repeat("r", 150_000)})
	cfg := &config.Config{
		ProjectRoot: root,
		Context:     config.ContextConfig{ToolResultMaxChars: 100_000},
	}
	r := &Runner{
		Tools:      reg,
		MCPResults: &resultstore.Store{ProjectRoot: root},
		Cfg:        cfg,
	}
	tc := llm.ToolCall{ID: "call_1", Name: "read_file"}
	body := formattedBody(strings.Repeat("r", 150_000))

	got := r.finalizeToolResult("sess-1", tc, body)
	if strings.Contains(got, "mcp-result") {
		t.Fatal("builtin tool should not get MCP spill hint")
	}
	base := datadir.DefaultMCPResultDir(root)
	if _, err := os.Stat(base); err == nil {
		t.Fatal("builtin tool should not create mcp-result dir")
	}
}

func TestFinalizeToolResult_mcpErrorNoSpill(t *testing.T) {
	root := t.TempDir()
	r, _ := mcpRunner(t, root, 100_000)
	tc := llm.ToolCall{ID: "call_1", Name: "mcp_tool"}
	body := toolresultpkg.FormatToolError("mcp_tool", "call_1", os.ErrPermission)

	got := r.finalizeToolResult("sess-1", tc, body)
	if strings.Contains(got, "mcp-result") {
		t.Fatal("error body should not spill")
	}
	base := datadir.DefaultMCPResultDir(root)
	if _, err := os.Stat(base); !os.IsNotExist(err) {
		t.Fatalf("mcp-result should not exist: %v", err)
	}
}

func TestFinalizeToolResult_mcpSuccessBodyStartsWithError(t *testing.T) {
	// Known limitation: bodies starting with "error:" skip spill (see DESIGN §12.5).
	body := formattedBody("error: but this is ok")
	if !isToolErrorBody(body) {
		t.Fatal("isToolErrorBody treats error: prefix as failure (known limitation)")
	}
	root := t.TempDir()
	r, _ := mcpRunner(t, root, 100_000)
	tc := llm.ToolCall{ID: "call_1", Name: "mcp_tool"}
	got := r.finalizeToolResult("sess-1", tc, body)
	base := datadir.DefaultMCPResultDir(root)
	if _, err := os.Stat(base); !os.IsNotExist(err) {
		t.Fatal("error:-prefixed success body should not spill")
	}
	_ = got
}

func extractHintPath(hint string) (string, bool) {
	const prefix = "保存至 "
	i := strings.Index(hint, prefix)
	if i < 0 {
		return "", false
	}
	rest := hint[i+len(prefix):]
	j := strings.Index(rest, "；")
	if j < 0 {
		return "", false
	}
	return rest[:j], true
}

func TestFinalizeToolResult_mcpHintPathReadable(t *testing.T) {
	root := t.TempDir()
	r, _ := mcpRunner(t, root, 100_000)
	r.Perm.ProjectRoot = root
	inner := strings.Repeat("m", 150_000)
	tc := llm.ToolCall{ID: "call_abc", Name: "mcp_tool"}

	got := r.finalizeToolResult("sess-1", tc, formattedBody(inner))
	path, ok := extractHintPath(got)
	if !ok {
		t.Fatalf("hint path not found in %q", got[len(got)-200:])
	}
	abs, err := r.Perm.CheckReadablePath(path)
	if err != nil {
		t.Fatalf("hint path not readable: %v", err)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), strings.Repeat("m", 100)) {
		t.Fatal("spill file should contain full MCP body")
	}
}

func TestFinalizeToolResult_mcpSpillBudgetZeroHintOnly(t *testing.T) {
	root := t.TempDir()
	testutil.IsolatedHome(t)
	store := &resultstore.Store{ProjectRoot: root}
	body := formattedBody(strings.Repeat("z", 10_000))
	path, err := store.Save("sess-1", "call_1", body)
	if err != nil {
		t.Fatal(err)
	}
	displayPath := toolresultpkg.ShortenSpillPathForHint(path, 100_000)
	hint := toolresultpkg.MCPSavedResultHint(displayPath)
	max := len(hint)

	r, _ := mcpRunner(t, root, max)
	r.Perm.ProjectRoot = root
	tc := llm.ToolCall{ID: "call_1", Name: "mcp_tool"}

	got := r.finalizeToolResult("sess-1", tc, body)
	if len(got) != max {
		t.Fatalf("len %d want hint-only %d", len(got), max)
	}
	readPath, ok := extractHintPath(got)
	if !ok {
		t.Fatalf("expected hint-only body: %q", got)
	}
	if _, err := r.Perm.CheckReadablePath(readPath); err != nil {
		t.Fatalf("hint path should be readable with budget=0: %v", err)
	}
}

func TestFinalizeToolResult_spillSaveFailed(t *testing.T) {
	root := t.TempDir()
	testutil.IsolatedHome(t)
	reg := tool.NewRegistry()
	reg.RegisterMCPTool(&stubMCPTool{name: "mcp_tool"}, "srv")
	cfg := &config.Config{
		ProjectRoot: root,
		Context:     config.ContextConfig{ToolResultMaxChars: 100},
	}
	// Block mkdir by making mcp-result a file.
	dataDir, err := datadir.ProjectDataDir(root)
	if err != nil {
		t.Fatal(err)
	}
	blocker := filepath.Join(dataDir, "mcp-result")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blocker, []byte("not a dir"), 0o600); err != nil {
		t.Fatal(err)
	}

	r := &Runner{Tools: reg, MCPResults: &resultstore.Store{ProjectRoot: root}, Cfg: cfg}
	inner := strings.Repeat("b", 200)
	tc := llm.ToolCall{ID: "call_1", Name: "mcp_tool"}

	got := r.finalizeToolResult("sess-1", tc, formattedBody(inner))
	if strings.Contains(got, "mcp-result/sess-1") {
		t.Fatal("failed save should not append spill hint path")
	}
	if !strings.Contains(got, "...") {
		t.Fatal("should fall back to generic truncation")
	}
}
