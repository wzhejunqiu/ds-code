package context_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/context"
	"github.com/hejunqiu/ds-code/internal/permission"
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

func TestAtExpander_dirSkipsSensitiveFiles(t *testing.T) {
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

	out, err := exp.Expand("see @pkg/")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "SECRET=") {
		t.Fatalf("directory @ ref leaked .env: %q", out)
	}
	if !strings.Contains(out, "package pkg") {
		t.Fatalf("expected pkg/code.go content: %q", out)
	}
}

func TestAtExpander_sensitiveDenied(t *testing.T) {
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
	if !strings.Contains(out, "error:") {
		t.Fatalf("expected permission error in output: %q", out)
	}
}
