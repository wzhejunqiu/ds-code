package shell_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/tool"
	"github.com/wzhejunqiu/ds-code/internal/tool/builtin/shell"
)

func newSyncShellTool(t *testing.T, dir string) *shell.ShellTool {
	t.Helper()
	return &shell.ShellTool{
		Cfg: &config.Config{
			ProjectRoot: dir,
			Tools: config.ToolsConfig{
				Shell: config.ShellToolConfig{Timeout: 120 * time.Second},
			},
		},
		Perm:   permission.NewEngine("auto", dir, false),
		Strict: false,
	}
}

func TestShellTool_nameAndPermissionLevel(t *testing.T) {
	st := &shell.ShellTool{}
	if st.Name() != tool.NameShell.String() {
		t.Fatalf("Name() = %q, want %q", st.Name(), tool.NameShell)
	}
	if st.PermissionLevel() != permission.LevelHighest {
		t.Fatalf("PermissionLevel() = %v, want LevelHighest", st.PermissionLevel())
	}
}

func TestShellTool_withPermRebindsEngine(t *testing.T) {
	parent := permission.NewEngine("auto", "/parent", false)
	child := permission.NewEngine("auto", "/child", false)
	st := &shell.ShellTool{Perm: parent}

	rebound, ok := st.WithPerm(child).(*shell.ShellTool)
	if !ok {
		t.Fatal("WithPerm should return *ShellTool")
	}
	if rebound.Perm.Workspace != "/child" {
		t.Fatalf("rebound workspace = %q, want /child", rebound.Perm.Workspace)
	}
	if st.Perm.Workspace != "/parent" {
		t.Fatalf("original tool should be unchanged, workspace = %q", st.Perm.Workspace)
	}
}

func TestShellTool_syncEchoStdout(t *testing.T) {
	dir := t.TempDir()
	st := newSyncShellTool(t, dir)
	args, _ := json.Marshal(map[string]any{"command": "echo hello-shell"})

	out, err := st.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "hello-shell") {
		t.Fatalf("expected stdout in output, got: %q", out)
	}
}

func TestShellTool_syncCapturesStderr(t *testing.T) {
	dir := t.TempDir()
	st := newSyncShellTool(t, dir)
	args, _ := json.Marshal(map[string]any{"command": "echo err-msg >&2"})

	out, err := st.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "stderr:") || !strings.Contains(out, "err-msg") {
		t.Fatalf("expected stderr in output, got: %q", out)
	}
}

func TestShellTool_syncNoOutput(t *testing.T) {
	dir := t.TempDir()
	st := newSyncShellTool(t, dir)
	args, _ := json.Marshal(map[string]any{"command": "true"})

	out, err := st.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if out != shell.ResultNoOutput {
		t.Fatalf("expected no output marker, got: %q", out)
	}
}

func TestShellTool_syncNonZeroExitCode(t *testing.T) {
	dir := t.TempDir()
	st := newSyncShellTool(t, dir)
	args, _ := json.Marshal(map[string]any{"command": "exit 7"})

	out, err := st.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, shell.ResultExitPrefix+"exit status 7") {
		t.Fatalf("expected exit status in output, got: %q", out)
	}
}

func TestShellTool_emptyCommandRejected(t *testing.T) {
	dir := t.TempDir()
	st := newSyncShellTool(t, dir)

	for _, raw := range []json.RawMessage{
		[]byte(`{"command":""}`),
		[]byte(`{"command":"   "}`),
		[]byte(`{}`),
	} {
		_, err := st.Execute(context.Background(), raw)
		if err == nil || !strings.Contains(err.Error(), shell.ErrCommandRequired) {
			t.Fatalf("args %s: expected %q, got err=%v", raw, shell.ErrCommandRequired, err)
		}
	}
}

func TestShellTool_invalidJSON(t *testing.T) {
	dir := t.TempDir()
	st := newSyncShellTool(t, dir)
	_, err := st.Execute(context.Background(), json.RawMessage("{bad"))
	if err == nil {
		t.Fatal("expected JSON unmarshal error")
	}
}

func TestShellTool_cancelledContext(t *testing.T) {
	dir := t.TempDir()
	st := newSyncShellTool(t, dir)
	args, _ := json.Marshal(map[string]any{"command": "echo hi"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := st.Execute(ctx, args)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestShellTool_backgroundUnavailableWithoutJobs(t *testing.T) {
	dir := t.TempDir()
	st := newSyncShellTool(t, dir)
	args, _ := json.Marshal(map[string]any{
		"command":           "echo hi",
		"run_in_background": true,
	})

	_, err := st.Execute(context.Background(), args)
	if err == nil || !strings.Contains(err.Error(), shell.ErrBackgroundUnavailable) {
		t.Fatalf("expected %q, got err=%v", shell.ErrBackgroundUnavailable, err)
	}
}

func TestShellTool_runsInWorkspace(t *testing.T) {
	dir := t.TempDir()
	marker := "workspace-marker.txt"
	if err := os.WriteFile(filepath.Join(dir, marker), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}

	st := newSyncShellTool(t, dir)
	args, _ := json.Marshal(map[string]any{"command": "ls"})

	out, err := st.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, marker) {
		t.Fatalf("expected %q in ls output, got: %q", marker, out)
	}
}

func TestShellTool_respectsShellEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SHELL", "/bin/sh")

	st := newSyncShellTool(t, dir)
	args, _ := json.Marshal(map[string]any{"command": "echo shell-env-ok"})

	out, err := st.Execute(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "shell-env-ok") {
		t.Fatalf("expected command output via $SHELL, got: %q", out)
	}
}

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
