package glob_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/glob"
	"github.com/wzhejunqiu/ds-code/internal/tool/searchskip"
)

type globFixture struct {
	dir string
	g   *glob.GlobTool
}

func newGlobFixture(t *testing.T, opts ...func(*config.Config)) *globFixture {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.Config{Tools: config.ToolsConfig{
		Glob: config.GlobToolConfig{
			MaxResults:       50,
			RespectGitignore: false,
			IncludeHidden:    true,
		},
		Grep: config.GrepToolConfig{
			Binary:  "bundled",
			Timeout: 20 * time.Second,
		},
		Search: config.SearchToolConfig{SkipDirs: []string{"node_modules"}},
	}}
	for _, opt := range opts {
		opt(cfg)
	}
	return &globFixture{
		dir: dir,
		g: &glob.GlobTool{
			Cfg:        cfg,
			Perm:       permission.NewEngine("readonly", dir, false),
			SearchSkip: searchskip.New(cfg.Tools.Search.SkipDirs),
		},
	}
}

func (f *globFixture) exec(t *testing.T, args map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	out, err := f.g.Execute(context.Background(), raw)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	return out
}

func (f *globFixture) write(t *testing.T, rel, content string) {
	t.Helper()
	p := filepath.Join(f.dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func (f *globFixture) chtimes(t *testing.T, rel string, mod time.Time) {
	t.Helper()
	p := filepath.Join(f.dir, filepath.FromSlash(rel))
	if err := os.Chtimes(p, mod, mod); err != nil {
		t.Fatal(err)
	}
}

func TestGlobTool_doubleStar(t *testing.T) {
	f := newGlobFixture(t)
	f.write(t, "a.go", "package a\n")
	f.write(t, "sub/b.go", "package b\n")
	out := f.exec(t, map[string]any{"pattern": "**/*.go"})
	if !strings.HasPrefix(out, "Found ") {
		t.Fatalf("expected Found summary, got %q", out)
	}
	if !strings.Contains(out, "a.go") {
		t.Fatalf("missing a.go: %q", out)
	}
	if !strings.Contains(out, "sub/b.go") {
		t.Fatalf("expected nested .go match: %q", out)
	}
}

func TestGlobTool_noMatch(t *testing.T) {
	f := newGlobFixture(t)
	f.write(t, "a.txt", "hello\n")
	out := f.exec(t, map[string]any{"pattern": "*.go"})
	if out != "Found 0 files" {
		t.Fatalf("got %q", out)
	}
}

func TestGlobTool_skipsSensitiveFiles(t *testing.T) {
	f := newGlobFixture(t)
	if err := os.WriteFile(filepath.Join(f.dir, ".env"), []byte("SECRET=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f.write(t, "ok.go", "package main\n")
	out := f.exec(t, map[string]any{"pattern": "**/*"})
	if strings.Contains(out, ".env") {
		t.Fatalf("glob leaked sensitive file: %q", out)
	}
	if !strings.Contains(out, "ok.go") {
		t.Fatalf("expected ok.go: %q", out)
	}
}

func TestGlobTool_skipsBinaryFiles(t *testing.T) {
	f := newGlobFixture(t)
	pngData := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	pngData = append(pngData, make([]byte, 32)...)
	if err := os.WriteFile(filepath.Join(f.dir, "img.png"), pngData, 0o644); err != nil {
		t.Fatal(err)
	}
	f.write(t, "main.go", "package main\n")
	out := f.exec(t, map[string]any{"pattern": "**/*"})
	if strings.Contains(out, "img.png") {
		t.Fatalf("glob should skip binary png: %q", out)
	}
	if !strings.Contains(out, "main.go") {
		t.Fatalf("expected main.go: %q", out)
	}
}

func TestGlobTool_ordersResultsByModTime(t *testing.T) {
	f := newGlobFixture(t)
	oldTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	newTime := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	f.write(t, "old.go", "package old\n")
	f.write(t, "new.go", "package new\n")
	f.chtimes(t, "old.go", oldTime)
	f.chtimes(t, "new.go", newTime)
	out := f.exec(t, map[string]any{"pattern": "*.go"})
	newIdx := strings.Index(out, "new.go")
	oldIdx := strings.Index(out, "old.go")
	if newIdx < 0 || oldIdx < 0 {
		t.Fatalf("expected both files: %q", out)
	}
	if newIdx > oldIdx {
		t.Fatalf("newer file should appear first: %q", out)
	}
}

func TestGlobTool_pathsRelativeToProjectRoot(t *testing.T) {
	f := newGlobFixture(t)
	f.write(t, "internal/pkg/a.go", "package pkg\n")
	out := f.exec(t, map[string]any{
		"pattern": "**/*.go",
		"path":    "internal/pkg",
	})
	want := "internal/pkg/a.go"
	if !strings.Contains(out, want) {
		t.Fatalf("expected workspace-relative path %q in output, got %q", want, out)
	}
	if strings.HasPrefix(strings.TrimSpace(strings.Split(out, "\n")[1]), "a.go") {
		t.Fatalf("should not be relative to search dir only: %q", out)
	}
}

func TestGlobTool_noAbsolutePathLeak(t *testing.T) {
	f := newGlobFixture(t)
	f.write(t, "a.go", "package a\n")
	out := f.exec(t, map[string]any{"pattern": "*.go"})
	if strings.Contains(out, f.dir) {
		t.Fatalf("absolute path leaked: %q", out)
	}
}

func TestGlobTool_explicitSkipDirPath(t *testing.T) {
	f := newGlobFixture(t)
	f.write(t, "node_modules/pkg/lib.go", "package lib\n")
	out := f.exec(t, map[string]any{"pattern": "**/*.go", "path": "node_modules"})
	if !strings.Contains(out, "node_modules/pkg/lib.go") {
		t.Fatalf("explicit path=node_modules should match under skip_dir: %q", out)
	}
	out = f.exec(t, map[string]any{"pattern": "**/*.go", "path": "."})
	if strings.Contains(out, "node_modules") {
		t.Fatalf("path=. should skip node_modules walk: %q", out)
	}
}

func TestGlobTool_pathGitEmpty(t *testing.T) {
	f := newGlobFixture(t)
	if err := os.MkdirAll(filepath.Join(f.dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.dir, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := f.exec(t, map[string]any{"pattern": "**/*", "path": ".git"})
	if out != "Found 0 files" {
		t.Fatalf("path=.git should return Found 0 files, got %q", out)
	}
}

func TestGlobTool_respectGitignoreTrue(t *testing.T) {
	f := newGlobFixture(t, func(c *config.Config) {
		c.Tools.Glob.RespectGitignore = true
	})
	f.write(t, ".gitignore", "ignored/\n")
	f.write(t, "ignored/hit.go", "package hit\n")
	if err := os.MkdirAll(filepath.Join(f.dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.dir, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := f.exec(t, map[string]any{"pattern": "**/*.go"})
	if strings.Contains(out, "ignored") {
		t.Fatalf("respect_gitignore=true should skip ignored: %q", out)
	}
}

func TestGlobTool_respectGitignoreFalse(t *testing.T) {
	f := newGlobFixture(t)
	f.write(t, ".gitignore", "ignored/\n")
	f.write(t, "ignored/hit.go", "package hit\n")
	out := f.exec(t, map[string]any{"pattern": "**/*.go", "path": "ignored"})
	if !strings.Contains(out, "hit.go") {
		t.Fatalf("default should search ignored when path explicit: %q", out)
	}
}

func TestGlobTool_includeHiddenDefault(t *testing.T) {
	f := newGlobFixture(t)
	if err := os.WriteFile(filepath.Join(f.dir, ".hidden"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := f.exec(t, map[string]any{"pattern": "**/*"})
	if !strings.Contains(out, ".hidden") {
		t.Fatalf("include_hidden=true should find hidden files: %q", out)
	}
}

func TestGlobTool_includeHiddenFalse(t *testing.T) {
	f := newGlobFixture(t, func(c *config.Config) {
		c.Tools.Glob.IncludeHidden = false
	})
	if err := os.WriteFile(filepath.Join(f.dir, ".hidden"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f.write(t, "visible.txt", "hi\n")
	out := f.exec(t, map[string]any{"pattern": "**/*"})
	if strings.Contains(out, ".hidden") {
		t.Fatalf("include_hidden=false should skip hidden: %q", out)
	}
	if !strings.Contains(out, "visible.txt") {
		t.Fatalf("expected visible.txt: %q", out)
	}
}

func TestGlobTool_explicitPathNotFound(t *testing.T) {
	f := newGlobFixture(t)
	raw, _ := json.Marshal(map[string]any{"pattern": "*.go", "path": "no/such/dir"})
	_, err := f.g.Execute(context.Background(), raw)
	if err == nil {
		t.Fatal("expected error for missing path")
	}
	if !strings.Contains(err.Error(), "目录不存在") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGlobTool_explicitPathFileRejected(t *testing.T) {
	f := newGlobFixture(t)
	f.write(t, "file.go", "package x\n")
	raw, _ := json.Marshal(map[string]any{"pattern": "*.go", "path": "file.go"})
	_, err := f.g.Execute(context.Background(), raw)
	if err == nil {
		t.Fatal("expected error for file path")
	}
	if !strings.Contains(err.Error(), "必须是目录") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGlobTool_maxResultsTruncated(t *testing.T) {
	f := newGlobFixture(t, func(c *config.Config) {
		c.Tools.Glob.MaxResults = 2
	})
	for i := 0; i < 5; i++ {
		f.write(t, fmt.Sprintf("f%02d.go", i), "package main\n")
	}
	out := f.exec(t, map[string]any{"pattern": "*.go"})
	if !strings.HasPrefix(out, "Found 2 files\n") {
		t.Fatalf("got %q", out)
	}
	if !strings.Contains(out, glob.MsgResultsTruncated) {
		t.Fatalf("missing truncation footer: %q", out)
	}
}

func TestGlobTool_contextCanceled(t *testing.T) {
	f := newGlobFixture(t)
	for i := 0; i < 100; i++ {
		f.write(t, fmt.Sprintf("f%03d.go", i), "package main\n")
	}
	raw, _ := json.Marshal(map[string]any{"pattern": "**/*.go"})
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := f.g.Execute(ctx, raw)
		errCh <- err
	}()
	cancel()
	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}
