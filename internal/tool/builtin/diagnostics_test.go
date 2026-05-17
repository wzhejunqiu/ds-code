package builtin

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/lsp"
	"github.com/hejunqiu/ds-code/internal/permission"
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
