package grep_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool"
	"github.com/hejunqiu/ds-code/internal/tool/builtin/grep"
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
	tool := &grep.GrepTool{
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

func TestGrepTool_respectsGitignoreInSubdirectory(t *testing.T) {
	dir := t.TempDir()
	pkg := filepath.Join(dir, "pkg")
	src := filepath.Join(pkg, "src")
	build := filepath.Join(pkg, "build")
	for _, p := range []string{pkg, src, build} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("pkg/build/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "hit.txt"), []byte("needle hit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(build, "miss.txt"), []byte("needle miss\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	gi, err := tool.LoadGitignore(dir)
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Tools: config.ToolsConfig{Grep: config.GrepToolConfig{HeadLimit: 50}}}
	grep := &grep.GrepTool{
		Cfg:       cfg,
		Perm:      permission.NewEngine("readonly", dir, false),
		Gitignore: gi,
	}
	args, _ := json.Marshal(map[string]any{"pattern": "needle", "path": "pkg"})
	out, err := grep.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "miss.txt") || strings.Contains(out, "needle miss") {
		t.Fatalf("grep should respect .gitignore under subdirectory search root: %q", out)
	}
	if !strings.Contains(out, "hit.txt") {
		t.Fatalf("expected match in pkg/src/hit.txt: %q", out)
	}
}
