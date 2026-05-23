package shell_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/shelljobs/manager"
	"github.com/hejunqiu/ds-code/internal/tool/builtin/shell"
)

func TestShellTool_backgroundAndPoll(t *testing.T) {
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

	startArgs, _ := json.Marshal(map[string]any{
		"command":    "echo tool-bg",
		"background": true,
	})
	out, err := tool.Execute(context.Background(), startArgs)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "job_id:") {
		t.Fatalf("start out: %s", out)
	}
	jobID := ""
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "job_id: ") {
			jobID = strings.TrimSpace(strings.TrimPrefix(line, "job_id:"))
		}
	}
	if jobID == "" {
		t.Fatal("missing job_id")
	}

	deadline := time.Now().Add(10 * time.Second)
	for {
		pollArgs, _ := json.Marshal(map[string]any{"job_id": jobID})
		poll, err := tool.Execute(context.Background(), pollArgs)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(poll, "status: completed") && strings.Contains(poll, "tool-bg") {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("poll timeout: %s", poll)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
