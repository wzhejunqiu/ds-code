package diagnostics

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/lsp"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool/searchskip"
)

func TestCollectFiles_skipsSensitiveUnderRoot(t *testing.T) {
	dir := t.TempDir()
	credDir := filepath.Join(dir, "credentials")
	if err := os.MkdirAll(credDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(credDir, "keys.go"), []byte("package credentials\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		ProjectRoot: dir,
		LSP:         config.LSPConfig{Enabled: true},
	}
	perm := permission.NewEngine("readonly", dir, false)
	tool := &DiagnosticsTool{
		Cfg:  cfg,
		Perm: perm,
		LSP:  lsp.NewManager(dir, cfg.LSP),
	}

	files, notes, err := tool.collectFiles(context.Background(), []string{"."}, 20)
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(files, "credentials/keys.go") {
		t.Fatalf("collectFiles included sensitive path: %v notes=%v", files, notes)
	}
	if !slices.Contains(files, "main.go") {
		t.Fatalf("expected main.go in %v", files)
	}
}

func TestCollectFiles_respectsSkipDirs(t *testing.T) {
	dir := t.TempDir()
	nm := filepath.Join(dir, "node_modules", "pkg")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nm, "lib.go"), []byte("package lib\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{ProjectRoot: dir, LSP: config.LSPConfig{Enabled: true}}
	perm := permission.NewEngine("readonly", dir, false)
	tool := &DiagnosticsTool{
		Cfg:        cfg,
		Perm:       perm,
		LSP:        lsp.NewManager(dir, cfg.LSP),
		SearchSkip: searchskip.New([]string{"node_modules"}),
	}

	files, _, err := tool.collectFiles(context.Background(), []string{"."}, 20)
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(files, "node_modules/pkg/lib.go") {
		t.Fatalf("path=. should filter skip_dirs: %v", files)
	}

	files, _, err = tool.collectFiles(context.Background(), []string{"node_modules/pkg"}, 20)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(files, "node_modules/pkg/lib.go") {
		t.Fatalf("explicit path should include skip_dir files: %v", files)
	}
}

func TestCollectFiles_alwaysFiltersGit(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{ProjectRoot: dir, LSP: config.LSPConfig{Enabled: true}}
	perm := permission.NewEngine("readonly", dir, false)
	tool := &DiagnosticsTool{
		Cfg:        cfg,
		Perm:       perm,
		LSP:        lsp.NewManager(dir, cfg.LSP),
		SearchSkip: searchskip.New(nil),
	}

	files, _, err := tool.collectFiles(context.Background(), []string{".git"}, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("path=.git should yield no diagnostic files, got %v", files)
	}
}
