package app

import (
	"context"
	"os"
	"strings"
	"testing"

	mcpsdk "github.com/mark3labs/mcp-go/mcp"
	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/logging"
	mcpsvc "github.com/wzhejunqiu/ds-code/internal/mcp"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/runmode"
	"github.com/wzhejunqiu/ds-code/internal/testutil"
)

func TestMCPSkipped_LoggedAfterBuildTools(t *testing.T) {
	testutil.IsolatedHome(t)
	root := t.TempDir()

	cleanup, err := logging.Setup(logging.Options{ProjectRoot: root, Verbosity: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	srv := mcpsvc.StubServer("fs", []mcpsdk.Tool{{Name: "grep", Description: "mcp grep"}})
	mgr := mcpsvc.NewManagerWithServers(srv)
	if err := mgr.DiscoverTools(context.Background(), false); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		ProjectRoot: root,
		RunMode:     runmode.Agent,
		LLM:         config.LLMConfig{Model: "test"},
		Permission:  config.PermissionConfig{Mode: "readonly"},
		LSP:         config.LSPConfig{Enabled: false},
	}
	a := New(cfg)
	a.mcpMgr = mgr

	perm := permission.NewEngine(cfg.Permission.Mode, root, false)
	if _, err := a.buildTools(context.Background(), perm, nil, false, nil, runmode.Agent); err != nil {
		t.Fatal(err)
	}
	a.logMCPSkipped()

	b, err := os.ReadFile(config.DefaultLogPath(root))
	if err != nil {
		t.Fatal(err)
	}
	body := string(b)
	if !strings.Contains(body, "mcp tool skipped") {
		t.Fatalf("log missing skip message: %q", body)
	}
	if !strings.Contains(body, "builtin_conflict") {
		t.Fatalf("log missing builtin_conflict: %q", body)
	}
}
