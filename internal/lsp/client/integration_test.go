//go:build integration

package client_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/lsp"
	"github.com/hejunqiu/ds-code/internal/testutil"
)

func TestClient_OpenFile_goplsReportsTypeError(t *testing.T) {
	testutil.RequireGopls(t)

	root := testutil.WriteGoModuleWithTypeError(t)
	content, err := os.ReadFile(filepath.Join(root, "main.go"))
	if err != nil {
		t.Fatal(err)
	}

	cfg := config.LSPConfig{
		Enabled:            true,
		DiagnosticsTimeout: 45 * time.Second,
		MaxIssuesPerFile:   20,
	}
	srv := lsp.BuildRegistry(cfg)["go"]
	client := client.NewClient(root, cfg, srv)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := client.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	sev := map[string]bool{"error": true, "warning": true}
	diags, err := client.OpenFile(ctx, "main.go", content, sev, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) == 0 {
		t.Fatal("expected at least one diagnostic from gopls")
	}
	found := false
	for _, d := range diags {
		if d.Severity == "error" || d.Severity == "warning" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("diagnostics: %+v", diags)
	}
}

func TestManager_EnsureClient_gopls(t *testing.T) {
	testutil.RequireGopls(t)

	root := testutil.WriteGoModuleWithTypeError(t)
	cfg := config.LSPConfig{
		Enabled:            true,
		DiagnosticsTimeout: 45 * time.Second,
	}
	mgr := lsp.NewManager(root, cfg)
	defer func() { _ = mgr.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := mgr.EnsureClient(ctx, "go")
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("nil client")
	}
}
