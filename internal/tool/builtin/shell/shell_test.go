package shell_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/shell"
)

func TestShellTool_schemaHasNewFields(t *testing.T) {
	tool := &shell.ShellTool{Strict: false}
	s := tool.Schema()
	props, _ := s["properties"].(map[string]any)
	for _, key := range []string{"run_in_background", "timeout_ms"} {
		if props[key] == nil {
			t.Fatalf("schema missing %q", key)
		}
	}
	for _, key := range []string{"background", "list_jobs", "job_id", "cancel"} {
		if props[key] != nil {
			t.Fatalf("schema should not contain %q", key)
		}
	}
}

func TestShellTool_timeoutMsKillsLongCommand(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		ProjectRoot: dir,
		Tools: config.ToolsConfig{
			Shell: config.ShellToolConfig{Timeout: 120 * time.Second},
		},
	}
	tool := &shell.ShellTool{Cfg: cfg, Perm: permission.NewEngine("auto", dir, false), Strict: false}

	args, _ := json.Marshal(map[string]any{
		"command":    "sleep 2",
		"timeout_ms": 200,
	})
	start := time.Now()
	out, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("expected timeout around 200ms, took %v", elapsed)
	}
	if !strings.Contains(out, "deadline exceeded") && !strings.Contains(out, "signal: killed") {
		t.Fatalf("expected timeout kill in output: %s", out)
	}
}

func TestShellTool_negativeTimeoutMsRejected(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{ProjectRoot: dir}
	tool := &shell.ShellTool{Cfg: cfg, Perm: permission.NewEngine("auto", dir, false), Strict: false}
	args, _ := json.Marshal(map[string]any{
		"command":    "echo hi",
		"timeout_ms": -1,
	})
	_, err := tool.Execute(context.Background(), args)
	if err == nil {
		t.Fatal("expected error for negative timeout_ms")
	}
}

func TestResolveSyncTimeout_capsAtTenMinutes(t *testing.T) {
	cfg := &config.Config{Tools: config.ToolsConfig{Shell: config.ShellToolConfig{Timeout: time.Minute}}}
	d, err := shell.ResolveSyncTimeout(cfg, 999_999_999)
	if err != nil {
		t.Fatal(err)
	}
	if d != 600*time.Second {
		t.Fatalf("got %v, want 10m", d)
	}
}

func TestShellTimeoutDeadline_syncAndBackground(t *testing.T) {
	cfg := &config.Config{Tools: config.ToolsConfig{Shell: config.ShellToolConfig{Timeout: 30 * time.Second}}}
	now := time.Now()

	syncArgs, _ := json.Marshal(map[string]any{"command": "echo hi"})
	deadline, ok := shell.ShellTimeoutDeadline(now, cfg, syncArgs)
	if !ok || !deadline.After(now) {
		t.Fatalf("sync deadline = %v ok=%v", deadline, ok)
	}

	bgArgs, _ := json.Marshal(map[string]any{"command": "echo hi", "run_in_background": true})
	bgDeadline, bgOK := shell.ShellTimeoutDeadline(now, cfg, bgArgs)
	if !bgOK || !bgDeadline.After(now) {
		t.Fatalf("background deadline = %v ok=%v", bgDeadline, bgOK)
	}
}
