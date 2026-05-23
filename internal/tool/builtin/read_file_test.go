package builtin_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool/builtin"
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
		"path":  "big.txt",
		"offset": 1,
		"limit": 10000,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "truncated to max_lines") {
		t.Fatalf("expected truncation hint: %q", out)
	}
}

func TestReadFile_maxBytesRejects(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "huge.txt"), []byte("xx"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := readFileTool(t, root, 500, 1)
	_, err := tool.Execute(context.Background(), mustJSON(t, map[string]any{"path": "huge.txt"}))
	if err == nil || !strings.Contains(err.Error(), "exceeds limit") {
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
	if !strings.Contains(out, "offset 5 beyond file length") {
		t.Fatalf("out = %q", out)
	}
}

func readFileTool(t *testing.T, root string, maxLines, maxBytes int) *builtin.ReadFileTool {
	t.Helper()
	cfg := &config.Config{
		ProjectRoot: root,
		Tools: config.ToolsConfig{
			ReadFile: config.ReadFileToolConfig{MaxLines: maxLines, MaxBytes: maxBytes},
		},
	}
	return &builtin.ReadFileTool{Cfg: cfg, Perm: permission.NewEngine("readonly", root, false)}
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
