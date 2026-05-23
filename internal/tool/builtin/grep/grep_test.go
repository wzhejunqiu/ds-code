package grep_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool"
	"github.com/hejunqiu/ds-code/internal/tool/builtin/grep"
)

func newGrepTool(t *testing.T, dir string, headLimit int, gi *tool.GitignoreMatcher) *grep.GrepTool {
	t.Helper()
	cfg := &config.Config{Tools: config.ToolsConfig{Grep: config.GrepToolConfig{HeadLimit: headLimit}}}
	return &grep.GrepTool{
		Cfg:       cfg,
		Perm:      permission.NewEngine("readonly", dir, false),
		Gitignore: gi,
	}
}

func TestGrepTool_skipsSensitiveFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=needle\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ok.txt"), []byte("needle here\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	g := newGrepTool(t, dir, 50, nil)
	args, _ := json.Marshal(map[string]any{"pattern": "needle"})
	out, err := g.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, ".env") || strings.Contains(out, "SECRET=") {
		t.Fatalf("grep leaked sensitive file: %q", out)
	}
	if !strings.Contains(out, "ok.txt") {
		t.Fatalf("expected match in ok.txt: %q", out)
	}
	if strings.Contains(out, ":1:") {
		t.Fatalf("default mode should be files_with_matches, got content: %q", out)
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

	g := newGrepTool(t, dir, 50, gi)
	args, _ := json.Marshal(map[string]any{"pattern": "needle", "path": "pkg"})
	out, err := g.Execute(context.Background(), args)
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

func TestGrepTool_ordersMatchesByFileModTime(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.txt")
	newPath := filepath.Join(dir, "new.txt")
	if err := os.WriteFile(oldPath, []byte("needle old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newPath, []byte("needle new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	newTime := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newPath, newTime, newTime); err != nil {
		t.Fatal(err)
	}

	g := newGrepTool(t, dir, 50, nil)
	args, _ := json.Marshal(map[string]any{"pattern": "needle", "output_mode": "content"})
	out, err := g.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	newIdx := strings.Index(out, "new.txt")
	oldIdx := strings.Index(out, "old.txt")
	if newIdx < 0 || oldIdx < 0 {
		t.Fatalf("expected both files in output: %q", out)
	}
	if newIdx > oldIdx {
		t.Fatalf("newer file should appear first: %q", out)
	}
}

func TestGrepTool_pathGlob(t *testing.T) {
	dir := t.TempDir()
	pkg := filepath.Join(dir, "pkg")
	other := filepath.Join(dir, "other")
	for _, p := range []string{pkg, other} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(pkg, "a.go"), []byte("needle in go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "b.txt"), []byte("needle in txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, "c.go"), []byte("no match\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	g := newGrepTool(t, dir, 50, nil)
	args, _ := json.Marshal(map[string]any{"pattern": "needle", "path": "pkg/*.go"})
	out, err := g.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "a.go") {
		t.Fatalf("expected a.go match: %q", out)
	}
	if strings.Contains(out, "b.txt") || strings.Contains(out, "other/c.go") {
		t.Fatalf("glob path should limit to pkg/*.go: %q", out)
	}
}

func TestGrepTool_skipsBinaryFiles(t *testing.T) {
	dir := t.TempDir()
	pngData := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	pngData = append(pngData, []byte("needle in png")...)
	if err := os.WriteFile(filepath.Join(dir, "img.png"), pngData, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ok.txt"), []byte("needle text\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	g := newGrepTool(t, dir, 50, nil)
	args, _ := json.Marshal(map[string]any{"pattern": "needle"})
	out, err := g.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "img.png") {
		t.Fatalf("grep should skip binary png: %q", out)
	}
	if !strings.Contains(out, "ok.txt") {
		t.Fatalf("expected ok.txt match: %q", out)
	}
}

func TestGrepTool_outputModeContent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("needle one\nneedle two\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	g := newGrepTool(t, dir, 50, nil)
	args, _ := json.Marshal(map[string]any{"pattern": "needle", "output_mode": "content"})
	out, err := g.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "a.txt:1:needle one") || !strings.Contains(out, "a.txt:2:needle two") {
		t.Fatalf("expected path:line:content: %q", out)
	}
}

func TestGrepTool_outputModeFilesWithMatches_dedupesFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("needle one\nneedle two\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	g := newGrepTool(t, dir, 50, nil)
	args, _ := json.Marshal(map[string]any{"pattern": "needle", "output_mode": "files_with_matches"})
	out, err := g.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if out != "a.txt" {
		t.Fatalf("expected single file path, got %q", out)
	}
}

func TestGrepTool_outputModeCount(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("needle one\nneedle two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("needle three\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	g := newGrepTool(t, dir, 50, nil)
	args, _ := json.Marshal(map[string]any{"pattern": "needle", "output_mode": "count"})
	out, err := g.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if out != "3" {
		t.Fatalf("expected total count 3, got %q", out)
	}
}

func TestGrepTool_outputModeCount_noMatches(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	g := newGrepTool(t, dir, 50, nil)
	args, _ := json.Marshal(map[string]any{"pattern": "needle", "output_mode": "count"})
	out, err := g.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if out != "0" {
		t.Fatalf("expected 0, got %q", out)
	}
}

func TestGrepTool_outputModeCount_ignoresHeadLimit(t *testing.T) {
	dir := t.TempDir()
	var b strings.Builder
	for i := 0; i < 10; i++ {
		b.WriteString("needle line\n")
	}
	if err := os.WriteFile(filepath.Join(dir, "many.txt"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	g := newGrepTool(t, dir, 3, nil)
	args, _ := json.Marshal(map[string]any{"pattern": "needle", "output_mode": "count"})
	out, err := g.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if out != "10" {
		t.Fatalf("count mode should ignore head_limit, got %q", out)
	}
}

func TestGrepTool_outputModeContent_headLimit(t *testing.T) {
	dir := t.TempDir()
	var b strings.Builder
	for i := 0; i < 5; i++ {
		b.WriteString("needle line\n")
	}
	if err := os.WriteFile(filepath.Join(dir, "many.txt"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	g := newGrepTool(t, dir, 2, nil)
	args, _ := json.Marshal(map[string]any{"pattern": "needle", "output_mode": "content"})
	out, err := g.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(out, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 2 matches + truncation line, got %d lines: %q", len(lines), out)
	}
	if !strings.Contains(lines[2], "已截断") {
		t.Fatalf("expected truncation suffix: %q", out)
	}
}

func TestGrepTool_invalidOutputMode(t *testing.T) {
	dir := t.TempDir()
	g := newGrepTool(t, dir, 50, nil)
	args, _ := json.Marshal(map[string]any{"pattern": "x", "output_mode": "invalid"})
	_, err := g.Execute(context.Background(), args)
	if err == nil {
		t.Fatal("expected error for invalid output_mode")
	}
}
