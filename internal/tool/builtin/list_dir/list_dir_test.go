package list_dir_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/list_dir"
	"github.com/wzhejunqiu/ds-code/internal/tool/searchskip"
)

func TestListDir_skipsSensitiveEntries(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{ProjectRoot: dir}
	perm := permission.NewEngine("readonly", dir, false)
	tool := &list_dir.ListDirTool{Cfg: cfg, Perm: perm, Strict: false}

	args, _ := json.Marshal(map[string]any{"path": "."})
	out, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, ".env") {
		t.Fatalf("list_dir should skip sensitive entries: %q", out)
	}
	if !strings.Contains(out, "readme.txt") {
		t.Fatalf("expected readme.txt: %q", out)
	}
}

func TestListDir_explicitSkipDirPath(t *testing.T) {
	dir := t.TempDir()
	nm := filepath.Join(dir, "node_modules", "pkg")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nm, "index.js"), []byte("// x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{ProjectRoot: dir, Tools: config.ToolsConfig{Glob: config.GlobToolConfig{MaxResults: 50}}}
	perm := permission.NewEngine("readonly", dir, false)
	tool := &list_dir.ListDirTool{Cfg: cfg, Perm: perm, SearchSkip: searchskip.New([]string{"node_modules"}), Strict: false}

	args, _ := json.Marshal(map[string]any{"path": "node_modules/pkg"})
	out, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "index.js") {
		t.Fatalf("explicit path=node_modules/pkg should list entries: %q", out)
	}
}
