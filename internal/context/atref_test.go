package context_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/context"
	"github.com/wzhejunqiu/ds-code/internal/permission"
)

func TestAtExpander_file(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "foo.go")
	if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Context: config.ContextConfig{AtReferenceMaxChars: 10000}}
	perm := permission.NewEngine("auto", dir, true)
	exp := &context.AtExpander{Cfg: cfg, Perm: perm}

	out, err := exp.Expand("explain @foo.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "package main") {
		t.Fatalf("output missing file content: %q", out)
	}
}

func TestAtExpander_dirIncludesSensitiveAtRoot(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "code.go"), []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Context: config.ContextConfig{AtReferenceMaxChars: 10000}}
	perm := permission.NewEngine("auto", dir, true)
	exp := &context.AtExpander{Cfg: cfg, Perm: perm}

	out, err := exp.Expand("see @./")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "SECRET=1") {
		t.Fatalf("directory @ ref should include .env when user explicitly @ ./: %q", out)
	}
	if !strings.Contains(out, "package pkg") {
		t.Fatalf("expected pkg/code.go content: %q", out)
	}
}

func TestAtExpander_sensitiveFileAllowed(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Context: config.ContextConfig{AtReferenceMaxChars: 10000}}
	perm := permission.NewEngine("auto", dir, true)
	exp := &context.AtExpander{Cfg: cfg, Perm: perm}

	out, err := exp.Expand("check @.env")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "SECRET=1") {
		t.Fatalf("expected @.env to expand sensitive content: %q", out)
	}
}

func TestAtExpander_sshDirAllowed(t *testing.T) {
	dir := t.TempDir()
	sshDir := filepath.Join(dir, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "config"), []byte("Host *\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Context: config.ContextConfig{AtReferenceMaxChars: 10000}}
	perm := permission.NewEngine("auto", dir, true)
	exp := &context.AtExpander{Cfg: cfg, Perm: perm}

	out, err := exp.Expand("review @.ssh/config")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Host *") {
		t.Fatalf("expected @.ssh/config content: %q", out)
	}
}

func TestAtExpander_dirIgnoresSkipDirs(t *testing.T) {
	dir := t.TempDir()
	nm := filepath.Join(dir, "node_modules", "pkg")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nm, "lib.go"), []byte("package lib\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Context: config.ContextConfig{AtReferenceMaxChars: 10000, AtDirMaxFiles: 20},
		Tools:   config.ToolsConfig{Search: config.SearchToolConfig{SkipDirs: []string{"node_modules"}}},
	}
	perm := permission.NewEngine("auto", dir, true)
	exp := &context.AtExpander{Cfg: cfg, Perm: perm}

	out, err := exp.Expand("load @node_modules/pkg/")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "package lib") {
		t.Fatalf("@dir should ignore skip_dirs: %q", out)
	}
}

func TestAtExpander_dirIgnoresGitignore(t *testing.T) {
	dir := t.TempDir()
	ignored := filepath.Join(dir, "ignored")
	if err := os.MkdirAll(ignored, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("ignored/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ignored, "code.go"), []byte("package ignored\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Context: config.ContextConfig{AtReferenceMaxChars: 10000, AtDirMaxFiles: 20}}
	perm := permission.NewEngine("auto", dir, true)
	exp := &context.AtExpander{Cfg: cfg, Perm: perm}

	out, err := exp.Expand("see @ignored/")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "package ignored") {
		t.Fatalf("@dir should ignore gitignore: %q", out)
	}
}

func TestAtExpander_dirAllowsNodeModules(t *testing.T) {
	dir := t.TempDir()
	nm := filepath.Join(dir, "node_modules", "left-pad")
	if err := os.MkdirAll(nm, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nm, "index.js"), []byte("module.exports = {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{Context: config.ContextConfig{AtReferenceMaxChars: 10000, AtDirMaxFiles: 20}}
	perm := permission.NewEngine("auto", dir, true)
	exp := &context.AtExpander{Cfg: cfg, Perm: perm}

	out, err := exp.Expand("inspect @node_modules/left-pad/")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "module.exports") {
		t.Fatalf("expected node_modules content: %q", out)
	}
}
