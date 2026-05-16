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

func TestGlobTool_doubleStar(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.go"), []byte("package b\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		ProjectRoot: dir,
		Tools:       config.ToolsConfig{Glob: config.GlobToolConfig{MaxResults: 50}},
	}
	perm := permission.NewEngine("readonly", dir, false)
	glob := &builtin.GlobTool{Cfg: cfg, Perm: perm, Strict: false}

	args, _ := json.Marshal(map[string]any{"pattern": "**/*.go"})
	out, err := glob.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "a.go") {
		t.Fatalf("missing a.go: %q", out)
	}
	if !strings.Contains(out, "b.go") && !strings.Contains(out, "sub") {
		t.Fatalf("expected nested .go match: %q", out)
	}
}

func TestListDirTool_basic(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		ProjectRoot: dir,
		Tools:       config.ToolsConfig{Glob: config.GlobToolConfig{MaxResults: 50}},
	}
	perm := permission.NewEngine("readonly", dir, false)
	list := &builtin.ListDirTool{Cfg: cfg, Perm: perm, Strict: false}

	args, _ := json.Marshal(map[string]any{"path": "."})
	out, err := list.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "readme.txt") || !strings.Contains(out, "pkg/") {
		t.Fatalf("unexpected listing: %q", out)
	}
}
