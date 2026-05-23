//go:build integration

package diagnostics_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/lsp"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/testutil"
	"github.com/hejunqiu/ds-code/internal/tool/builtin/diagnostics"
)

func TestDiagnosticsTool_goplsOnTypeErrorModule(t *testing.T) {
	testutil.RequireGopls(t)

	root := testutil.WriteGoModuleWithTypeError(t)
	cfg := &config.Config{
		ProjectRoot: root,
		LSP: config.LSPConfig{
			Enabled:            true,
			DiagnosticsTimeout: 45 * time.Second,
			MaxFilesPerCall:    5,
			MaxIssuesPerFile:   20,
		},
		Context: config.ContextConfig{ToolResultMaxChars: 100000},
	}
	perm := permission.NewEngine("readonly", root, false)
	mgr := lsp.NewManager(root, cfg.LSP)
	defer func() { _ = mgr.Close() }()

	diag := &diagnostics.DiagnosticsTool{
		Cfg:    cfg,
		Perm:   perm,
		LSP:    mgr,
		Strict: false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	args, _ := json.Marshal(map[string]any{
		"paths":    []string{"main.go"},
		"severity": []string{"error", "warning"},
	})
	out, err := diag.Execute(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
	if out == "" || out == "No diagnostics." {
		t.Fatalf("unexpected output: %q", out)
	}
	lower := strings.ToLower(out)
	if !strings.Contains(lower, "error") && !strings.Contains(lower, "cannot") && !strings.Contains(lower, "mismatch") {
		t.Fatalf("expected diagnostic text, got: %s", out)
	}
}
