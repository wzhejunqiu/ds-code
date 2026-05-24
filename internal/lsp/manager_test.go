package lsp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
)

var fakeLSPBin string

func TestMain(m *testing.M) {
	root := moduleRoot()
	out := filepath.Join(os.TempDir(), "ds-code-fakelsp-test")
	cmd := exec.Command("go", "build", "-o", out, "./internal/lsp/testdata/fakeserver")
	cmd.Dir = root
	if outBytes, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build fakeserver: %v\n%s", err, outBytes)
		os.Exit(1)
	}
	fakeLSPBin = out
	os.Exit(m.Run())
}

func buildFakeLSP(t *testing.T) string {
	t.Helper()
	return fakeLSPBin
}

func moduleRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("go.mod not found")
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
		DiagnosticsTimeout: 15 * time.Second,
		Servers: map[string]config.LSPServerConfig{
			"fake": {Command: bin},
		},
	}
	mgr := NewManager(root, cfg)
	defer func() { _ = mgr.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
