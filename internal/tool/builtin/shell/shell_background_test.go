package shell_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/shelljobs/manager"
	"github.com/wzhejunqiu/ds-code/internal/testutil"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/shell"
)

func TestShellTool_backgroundBlocksUntilComplete(t *testing.T) {
	testutil.IsolatedHome(t)
	dir := t.TempDir()
	cfg := &config.Config{
		ProjectRoot: dir,
		Tools: config.ToolsConfig{
			Shell: config.ShellToolConfig{
				MaxBackground:            2,
				BackgroundOutputMaxBytes: 65536,
			},
		},
	}
	mgr, err := manager.Open(dir, cfg.Tools.Shell)
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()

	tool := &shell.ShellTool{Cfg: cfg, Perm: permission.NewEngine("auto", dir, false), Jobs: mgr, Strict: false}

	args, _ := json.Marshal(map[string]any{
		"description":       "Echo background",
		"command":           "echo tool-bg",
		"run_in_background": true,
	})
	out, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "tool-bg") {
		t.Fatalf("expected output containing tool-bg, got: %s", out)
	}
	if strings.Contains(out, "job_id:") {
		t.Fatalf("should not return job_id, got: %s", out)
	}
}

func TestShellTool_backgroundStoresDescription(t *testing.T) {
	testutil.IsolatedHome(t)
	dir := t.TempDir()
	cfg := &config.Config{
		ProjectRoot: dir,
		Tools:       config.ToolsConfig{Shell: config.ShellToolConfig{MaxBackground: 2}},
	}
	mgr, err := manager.Open(dir, cfg.Tools.Shell)
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()

	tool := &shell.ShellTool{Cfg: cfg, Perm: permission.NewEngine("auto", dir, false), Jobs: mgr, Strict: false}
	args, _ := json.Marshal(map[string]any{
		"description":       "Run unit tests",
		"command":           "sleep 0.1",
		"run_in_background": true,
	})
	if _, err := tool.Execute(context.Background(), args); err != nil {
		t.Fatal(err)
	}
	jobs, err := mgr.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, j := range jobs {
		if j.Description == "Run unit tests" {
			return
		}
	}
	t.Fatalf("no job with description %q in %v", "Run unit tests", jobs)
}

func TestShellTool_backgroundTimeout(t *testing.T) {
	testutil.IsolatedHome(t)
	dir := t.TempDir()
	cfg := &config.Config{
		ProjectRoot: dir,
		Tools: config.ToolsConfig{
			Shell: config.ShellToolConfig{MaxBackground: 2, Timeout: 120 * time.Second},
		},
	}
	mgr, err := manager.Open(dir, cfg.Tools.Shell)
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()

	tool := &shell.ShellTool{Cfg: cfg, Perm: permission.NewEngine("auto", dir, false), Jobs: mgr, Strict: false}
	args, _ := json.Marshal(map[string]any{
		"command":           "sh -c 'echo partial; sleep 5'",
		"run_in_background": true,
		"timeout_ms":        200,
	})
	start := time.Now()
	out, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("expected timeout around 200ms, took %v", elapsed)
	}
	if !strings.Contains(out, "partial") {
		t.Fatalf("expected partial stdout before timeout, got: %s", out)
	}
	if !strings.Contains(out, "deadline exceeded") && !strings.Contains(out, "signal: killed") {
		t.Fatalf("expected timeout kill in output: %s", out)
	}
}
