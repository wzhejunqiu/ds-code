package lsp

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hejunqiu/ds-code/internal/config"
)

func buildFakeLSP(t *testing.T) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "fakelsp")
	cmd := exec.Command("go", "build", "-o", out, "./internal/lsp/testdata/fakeserver")
	cmd.Dir = moduleRoot(t)
	if outBytes, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fakeserver: %v\n%s", err, outBytes)
	}
	return out
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func TestManager_EnsureClient_errors(t *testing.T) {
	root := t.TempDir()
	mgr := NewManager(root, config.LSPConfig{})

	ctx := context.Background()
	if _, err := mgr.EnsureClient(ctx, "nope"); err == nil {
		t.Fatal("expected unknown server error")
	} else if !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("err = %v", err)
	}

	reg := BuildRegistry(config.LSPConfig{})
	java := reg["java"]
	java.Disabled = true
	reg["java"] = java
	mgr.registry = reg
	if _, err := mgr.EnsureClient(ctx, "java"); err == nil {
		t.Fatal("expected disabled error")
	}

	goSrv := reg["go"]
	goSrv.Command = ""
	reg["go"] = goSrv
	mgr.registry = reg
	if _, err := mgr.EnsureClient(ctx, "go"); err == nil {
		t.Fatal("expected no command error")
	}

	goSrv = reg["go"]
	goSrv.Command = "definitely-not-a-real-lsp-binary-xyz-12345"
	goSrv.Disabled = false
	reg["go"] = goSrv
	mgr.registry = reg
	if _, err := mgr.EnsureClient(ctx, "go"); err == nil {
		t.Fatal("expected not found in PATH error")
	}
}

func TestManager_EnsureClient_fakeServer(t *testing.T) {
	root := t.TempDir()
	bin := buildFakeLSP(t)
	cfg := config.LSPConfig{
		DiagnosticsTimeout: 5 * time.Second,
		Servers: map[string]config.LSPServerConfig{
			"fake": {Command: bin},
		},
	}
	mgr := NewManager(root, cfg)
	defer func() { _ = mgr.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c1, err := mgr.EnsureClient(ctx, "fake")
	if err != nil {
		t.Fatal(err)
	}
	c2, err := mgr.EnsureClient(ctx, "fake")
	if err != nil {
		t.Fatal(err)
	}
	if c1 != c2 {
		t.Fatal("expected cached client instance")
	}
}

func TestManager_OpenFile_viaFakeServer(t *testing.T) {
	root := t.TempDir()
	mainGo := filepath.Join(root, "main.go")
	if err := os.WriteFile(mainGo, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	bin := buildFakeLSP(t)
	cfg := config.LSPConfig{
		DiagnosticsTimeout: 3 * time.Second,
		Servers: map[string]config.LSPServerConfig{
			"fake": {Command: bin},
		},
	}
	mgr := NewManager(root, cfg)
	defer func() { _ = mgr.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mgr.EnsureClient(ctx, "fake")
	if err != nil {
		t.Fatal(err)
	}

	diags, err := client.OpenFile(ctx, "main.go", []byte("package main\n"), map[string]bool{"error": true}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) == 0 {
		t.Fatal("expected diagnostics from fake LSP server")
	}
	if diags[0].Message != "fake diagnostic" {
		t.Fatalf("diag = %+v", diags[0])
	}
}

func TestManager_Close(t *testing.T) {
	mgr := NewManager(t.TempDir(), config.LSPConfig{})
	if err := mgr.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestManager_Registry(t *testing.T) {
	cfg := config.LSPConfig{
		Servers: map[string]config.LSPServerConfig{
			"custom": {Command: "/bin/echo"},
		},
	}
	mgr := NewManager(t.TempDir(), cfg)
	reg := mgr.Registry()
	if reg["custom"].Command != "/bin/echo" {
		t.Fatalf("custom = %+v", reg["custom"])
	}
}
