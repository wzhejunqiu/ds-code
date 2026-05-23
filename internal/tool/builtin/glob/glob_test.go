package glob_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool/builtin/glob"
	"github.com/hejunqiu/ds-code/internal/tool/builtin/list_dir"
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
	glob := &glob.GlobTool{Cfg: cfg, Perm: perm, Strict: false}

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
	list := &list_dir.ListDirTool{Cfg: cfg, Perm: perm, Strict: false}

	args, _ := json.Marshal(map[string]any{"path": "."})
	out, err := list.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "readme.txt") || !strings.Contains(out, "pkg/") {
		t.Fatalf("unexpected listing: %q", out)
	}
}

func TestGlobTool_skipsSensitiveFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ok.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		ProjectRoot: dir,
		Tools:       config.ToolsConfig{Glob: config.GlobToolConfig{MaxResults: 50}},
	}
	perm := permission.NewEngine("readonly", dir, false)
	glob := &glob.GlobTool{Cfg: cfg, Perm: perm, Strict: false}

	args, _ := json.Marshal(map[string]any{"pattern": "**/*"})
	out, err := glob.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, ".env") {
		t.Fatalf("glob leaked sensitive file: %q", out)
	}
	if !strings.Contains(out, "ok.go") {
		t.Fatalf("expected ok.go: %q", out)
	}
}

func TestGlobTool_rejectsOutsideWorkspace(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "escape")
	if err := os.Symlink(secret, link); err != nil {
		t.Skip("symlink not supported")
	}

	cfg := &config.Config{ProjectRoot: dir, Tools: config.ToolsConfig{Glob: config.GlobToolConfig{MaxResults: 50}}}
	perm := permission.NewEngine("readonly", dir, false)
	glob := &glob.GlobTool{Cfg: cfg, Perm: perm, Strict: false}

	args, _ := json.Marshal(map[string]any{"pattern": "escape"})
	_, err := glob.Execute(context.Background(), args)
	if err == nil {
		t.Fatal("expected error for symlink outside workspace")
	}
	if !strings.Contains(err.Error(), "outside workspace") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGlobTool_skipsBinaryFiles(t *testing.T) {
	dir := t.TempDir()
	pngData := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	pngData = append(pngData, make([]byte, 32)...)
	if err := os.WriteFile(filepath.Join(dir, "img.png"), pngData, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		ProjectRoot: dir,
		Tools:       config.ToolsConfig{Glob: config.GlobToolConfig{MaxResults: 50}},
	}
	perm := permission.NewEngine("readonly", dir, false)
	globTool := &glob.GlobTool{Cfg: cfg, Perm: perm, Strict: false}

	args, _ := json.Marshal(map[string]any{"pattern": "**/*"})
	out, err := globTool.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "img.png") {
		t.Fatalf("glob should skip binary png: %q", out)
	}
	if !strings.Contains(out, "main.go") {
		t.Fatalf("expected main.go: %q", out)
	}
}

func TestListDirTool_skipsSensitiveEntries(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		ProjectRoot: dir,
		Tools:       config.ToolsConfig{Glob: config.GlobToolConfig{MaxResults: 50}},
	}
	perm := permission.NewEngine("readonly", dir, false)
	list := &list_dir.ListDirTool{Cfg: cfg, Perm: perm, Strict: false}

	args, _ := json.Marshal(map[string]any{"path": "."})
	out, err := list.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, ".env") {
		t.Fatalf("list_dir leaked .env: %q", out)
	}
	if !strings.Contains(out, "readme.txt") {
		t.Fatalf("expected readme.txt: %q", out)
	}
}
