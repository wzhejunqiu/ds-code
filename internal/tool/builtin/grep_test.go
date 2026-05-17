package builtin_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool/builtin"
)

func TestGrepTool_skipsSensitiveFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=needle\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ok.txt"), []byte("needle here\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Tools: config.ToolsConfig{Grep: config.GrepToolConfig{HeadLimit: 50}}}
	tool := &builtin.GrepTool{
		Cfg:  cfg,
		Perm: permission.NewEngine("readonly", dir, false),
	}
	args, _ := json.Marshal(map[string]any{"pattern": "needle"})
	out, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, ".env") || strings.Contains(out, "SECRET=") {
		t.Fatalf("grep leaked sensitive file: %q", out)
	}
	if !strings.Contains(out, "ok.txt") {
		t.Fatalf("expected match in ok.txt: %q", out)
	}
}
