package read_file_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/datadir"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	"github.com/wzhejunqiu/ds-code/internal/mcp/resultstore"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/testutil"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/read_file"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestReadFile_offsetLimit(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "f.txt")
	lines := make([]string, 30)
	for i := range lines {
		lines[i] = fmt.Sprintf("line%d", i+1)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := readFileTool(t, root, 500, 1<<20)
	out, err := tool.Execute(context.Background(), mustJSON(t, map[string]any{
		"path":   "f.txt",
		"offset": 10,
		"limit":  11,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "10|line10") || !strings.Contains(out, "20|line20") {
		t.Fatalf("output = %q", out)
	}
	if strings.Contains(out, "\n9|") || strings.Contains(out, "\n21|") {
		t.Fatalf("unexpected lines: %q", out)
	}
}

func TestReadFile_defaultReadsWholeFile(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	for i := 1; i <= 100; i++ {
		fmt.Fprintf(&b, "L%d\n", i)
	}
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := readFileTool(t, root, 2000, 1<<20)
	out, err := tool.Execute(context.Background(), mustJSON(t, map[string]any{"path": "f.txt"}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "1|L1") || !strings.Contains(out, "100|L100") {
		t.Fatalf("expected full file: %q", out)
	}
	if strings.Contains(out, "未显示") || strings.Contains(out, "max_lines") {
		t.Fatalf("unexpected truncation hint: %q", out)
	}
}

func TestReadFile_defaultTruncatesAtMaxLines(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	for i := 1; i <= 2500; i++ {
		fmt.Fprintf(&b, "L%d\n", i)
	}
	if err := os.WriteFile(filepath.Join(root, "big.txt"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := readFileTool(t, root, 2000, 1<<20)
	out, err := tool.Execute(context.Background(), mustJSON(t, map[string]any{"path": "big.txt"}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "2000|L2000") {
		t.Fatalf("expected line 2000: %q", out)
	}
	if strings.Contains(out, "2500|") {
		t.Fatalf("should not include line 2500: %q", out)
	}
	if !strings.Contains(out, "还有 500 行未显示") {
		t.Fatalf("expected more-lines hint: %q", out)
	}
}

func TestReadFile_maxLinesTruncatesLimit(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	for i := 1; i <= 100; i++ {
		fmt.Fprintf(&b, "L%d\n", i)
	}
	if err := os.WriteFile(filepath.Join(root, "big.txt"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := readFileTool(t, root, 10, 1<<20)
	out, err := tool.Execute(context.Background(), mustJSON(t, map[string]any{
		"path":   "big.txt",
		"offset": 1,
		"limit":  10000,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "已按 10 行截断") {
		t.Fatalf("expected truncation hint: %q", out)
	}
	if !strings.Contains(out, "还有 90 行未显示") {
		t.Fatalf("expected more-lines hint: %q", out)
	}
}

func TestReadFile_maxBytesRejects(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "huge.txt"), []byte("xx"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := readFileTool(t, root, 500, 1)
	_, err := tool.Execute(context.Background(), mustJSON(t, map[string]any{"path": "huge.txt"}))
	if err == nil || !strings.Contains(err.Error(), "超过上限") {
		t.Fatalf("err = %v", err)
	}
}

func TestReadFile_offsetBeyondFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tool := readFileTool(t, root, 500, 1<<20)
	out, err := tool.Execute(context.Background(), mustJSON(t, map[string]any{
		"path": "f.txt", "offset": 5,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "offset 5") {
		t.Fatalf("out = %q", out)
	}
}

func TestReadFile_rejectsNonText(t *testing.T) {
	root := t.TempDir()
	// Minimal PNG header bytes.
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	if err := os.WriteFile(filepath.Join(root, "img.png"), png, 0o644); err != nil {
		t.Fatal(err)
	}
	tool := readFileTool(t, root, 500, 1<<20)
	_, err := tool.Execute(context.Background(), mustJSON(t, map[string]any{"path": "img.png"}))
	if err == nil || !strings.Contains(err.Error(), "非文本") {
		t.Fatalf("err = %v", err)
	}
}

func TestReadFile_allowsEmptyFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "empty.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	tool := readFileTool(t, root, 500, 1<<20)
	out, err := tool.Execute(context.Background(), mustJSON(t, map[string]any{"path": "empty.txt"}))
	if err != nil {
		t.Fatal(err)
	}
	if out != "" && !strings.Contains(out, "超出文件长度 0") {
		t.Fatalf("out = %q", out)
	}
}

func TestReadFile_allowsMCPSpill(t *testing.T) {
	root := t.TempDir()
	testutil.IsolatedHome(t)
	store := &resultstore.Store{ProjectRoot: root}
	spillBody := strings.Repeat("mcp-full-", 100)
	spillPath, err := store.Save("sess-read", "call_1", spillBody)
	if err != nil {
		t.Fatal(err)
	}

	perm := permission.NewEngine("auto", root, false)
	perm.ProjectRoot = root
	cfg := &config.Config{
		ProjectRoot: root,
		Tools:       config.ToolsConfig{ReadFile: config.ReadFileToolConfig{MaxLines: 500, MaxBytes: 1 << 20}},
	}
	tool := &read_file.ReadFileTool{Cfg: cfg, Perm: perm, Strict: false}

	out, err := tool.Execute(context.Background(), mustJSON(t, map[string]any{"path": spillPath}))
	if err != nil {
		t.Fatalf("read spill: %v", err)
	}
	if !strings.Contains(out, "mcp-full-") {
		t.Fatalf("expected spill content: %q", out)
	}
}

func TestReadFile_allowsAgentsOutput(t *testing.T) {
	root := t.TempDir()
	testutil.IsolatedHome(t)
	perm := permission.NewEngine("auto", root, false)
	perm.ProjectRoot = root

	dataDir, err := datadir.ProjectDataDir(root)
	if err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(dataDir, "agents", "sess", "tc.output")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o700); err != nil {
		t.Fatal(err)
	}
	body := "status: completed\n\nagent summary line"
	if err := os.WriteFile(outPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		ProjectRoot: root,
		Tools:       config.ToolsConfig{ReadFile: config.ReadFileToolConfig{MaxLines: 500, MaxBytes: 1 << 20}},
	}
	tool := &read_file.ReadFileTool{Cfg: cfg, Perm: perm, Strict: false}
	out, err := tool.Execute(context.Background(), mustJSON(t, map[string]any{"path": outPath}))
	if err != nil {
		t.Fatalf("read agents output: %v", err)
	}
	if !strings.Contains(out, "agent summary line") {
		t.Fatalf("expected spill content: %q", out)
	}
}

func TestReadFile_allowsProjectDataDB(t *testing.T) {
	root := t.TempDir()
	testutil.IsolatedHome(t)
	perm := permission.NewEngine("auto", root, false)
	perm.ProjectRoot = root

	dataDir, err := datadir.ProjectDataDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dataDir, "sessions.db")
	if err := os.WriteFile(dbPath, []byte("sqlite-header"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		ProjectRoot: root,
		Tools:       config.ToolsConfig{ReadFile: config.ReadFileToolConfig{MaxLines: 500, MaxBytes: 1 << 20}},
	}
	tool := &read_file.ReadFileTool{Cfg: cfg, Perm: perm, Strict: false}
	out, err := tool.Execute(context.Background(), mustJSON(t, map[string]any{"path": dbPath}))
	if err != nil {
		t.Fatalf("read sessions.db: %v", err)
	}
	if !strings.Contains(out, "sqlite-header") {
		t.Fatalf("expected db content: %q", out)
	}
}

func TestReadFile_nonTextLogsInfo(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	restore := logging.ReplaceForTest(zap.New(core))
	defer restore()

	root := t.TempDir()
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	if err := os.WriteFile(filepath.Join(root, "img.png"), png, 0o644); err != nil {
		t.Fatal(err)
	}
	tool := readFileTool(t, root, 500, 1<<20)
	_, _ = tool.Execute(context.Background(), mustJSON(t, map[string]any{"path": "img.png"}))

	entries := logs.FilterMessage("read_file skipped non-text file")
	if entries.Len() != 1 {
		t.Fatalf("expected info log, got %d", entries.Len())
	}
	ctx := entries.All()[0].ContextMap()
	if ctx["path"] != "img.png" {
		t.Fatalf("path field = %v", ctx["path"])
	}
	if ctx["abs"] == nil || ctx["abs"] == "" {
		t.Fatal("expected abs field in log")
	}
}

func readFileTool(t *testing.T, root string, maxLines, maxBytes int) *read_file.ReadFileTool {
	t.Helper()
	cfg := &config.Config{
		ProjectRoot: root,
		Tools: config.ToolsConfig{
			ReadFile: config.ReadFileToolConfig{MaxLines: maxLines, MaxBytes: maxBytes},
		},
	}
	return &read_file.ReadFileTool{Cfg: cfg, Perm: permission.NewEngine("readonly", root, false)}
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
