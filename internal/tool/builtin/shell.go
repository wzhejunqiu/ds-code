package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/tool"
)

// ShellTool runs a shell command synchronously in the workspace.
type ShellTool struct {
	Cfg    *config.Config
	Perm   *permission.Engine
	Strict bool
}

func (t *ShellTool) Name() string { return "shell" }

func (t *ShellTool) Description() string {
	return "Run a shell command in the project workspace directory. Subject to permission mode."
}

func (t *ShellTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"command": map[string]any{"type": "string", "description": "Shell command to execute"},
	}, []string{"command"}, t.Strict)
}

func (t *ShellTool) PermissionLevel() permission.Level { return permission.LevelHighest }

func (t *ShellTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}
	if strings.TrimSpace(in.Command) == "" {
		return "", fmt.Errorf("command is required")
	}

	timeout := t.Cfg.Tools.Shell.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	cmd := exec.CommandContext(runCtx, shell, "-c", in.Command)
	cmd.Dir = t.Perm.Workspace
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	var b strings.Builder
	if stdout.Len() > 0 {
		b.WriteString("stdout:\n")
		b.Write(stdout.Bytes())
	}
	if stderr.Len() > 0 {
		b.WriteString("stderr:\n")
		b.Write(stderr.Bytes())
	}
	if err != nil {
		b.WriteString("exit: ")
		b.WriteString(err.Error())
	}
	if b.Len() == 0 {
		return "(no output)", nil
	}
	return b.String(), nil
}

var _ tool.Tool = (*ShellTool)(nil)
