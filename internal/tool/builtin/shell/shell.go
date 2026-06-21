package shell

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/permission"
	"github.com/wzhejunqiu/ds-code/internal/security"
	"github.com/wzhejunqiu/ds-code/internal/shelljobs"
	"github.com/wzhejunqiu/ds-code/internal/shelljobs/manager"
	"github.com/wzhejunqiu/ds-code/internal/tool"
)

// ShellTool runs shell commands synchronously or in the background.
type ShellTool struct {
	Cfg    *config.Config
	Perm   *permission.Engine
	Jobs   *manager.Manager
	Strict bool
}

func (t *ShellTool) Name() string { return tool.NameShell.String() }

func (t *ShellTool) WithPerm(perm *permission.Engine) tool.Tool {
	cp := *t
	cp.Perm = perm
	return &cp
}

func (t *ShellTool) Description() string { return RenderDesc() }

func (t *ShellTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"description": map[string]any{
			"type":        "string",
			"description": SchemaShellDescription,
		},
		"command": map[string]any{
			"type":        "string",
			"description": SchemaShellCommand,
		},
		"run_in_background": map[string]any{
			"type":        "boolean",
			"description": SchemaRunInBackground,
		},
		"timeout_ms": map[string]any{
			"type":        "integer",
			"description": SchemaTimeoutMs,
		},
	}, nil, t.Strict)
}

func (t *ShellTool) PermissionLevel() permission.Level { return permission.LevelHighest }

func (t *ShellTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var in struct {
		Description     string `json:"description"`
		Command         string `json:"command"`
		RunInBackground bool   `json:"run_in_background"`
		TimeoutMs       int    `json:"timeout_ms"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}

	if t.Jobs == nil {
		if in.RunInBackground {
			return "", fmt.Errorf("%s", ErrBackgroundUnavailable)
		}
		return t.runSync(ctx, in.Command, in.TimeoutMs)
	}

	if in.RunInBackground {
		return t.runBackground(ctx, in.Command, in.Description, in.TimeoutMs)
	}
	if strings.TrimSpace(in.Command) == "" {
		return "", fmt.Errorf("%s", ErrCommandRequired)
	}
	return t.runSync(ctx, in.Command, in.TimeoutMs)
}

func (t *ShellTool) runBackground(ctx context.Context, command, description string, timeoutMs int) (string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", fmt.Errorf("%s", ErrCommandRequired)
	}

	timeout, err := ResolveShellTimeout(t.Cfg, timeoutMs)
	if err != nil {
		return "", err
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	job, err := t.Jobs.Start(command, description)
	if err != nil {
		return "", err
	}
	finished, waitErr := t.Jobs.Wait(runCtx, job.ID)
	if waitErr != nil {
		return t.formatFinishedJob(job, waitErr)
	}
	return t.formatFinishedJob(finished, nil)
}

func (t *ShellTool) formatFinishedJob(job shelljobs.Job, waitErr error) (string, error) {
	maxBytes := t.Cfg.Tools.Shell.BackgroundOutputMaxBytes
	if maxBytes <= 0 {
		maxBytes = 256 * 1024
	}
	_, stdout, stderr, err := t.Jobs.Get(job.ID, maxBytes)
	if err != nil {
		return "", err
	}
	var runErr error
	if waitErr != nil {
		runErr = waitErr
	} else if job.Status == shelljobs.StatusKilled {
		runErr = fmt.Errorf("signal: killed")
	} else if job.Status == shelljobs.StatusFailed {
		if job.ExitCode != nil {
			runErr = fmt.Errorf("exit status %d", *job.ExitCode)
		} else {
			runErr = fmt.Errorf("command failed")
		}
	}
	return formatShellOutput(stdout, stderr, runErr)
}

func (t *ShellTool) runSync(ctx context.Context, command string, timeoutMs int) (string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", fmt.Errorf("%s", ErrCommandRequired)
	}

	timeout, err := ResolveShellTimeout(t.Cfg, timeoutMs)
	if err != nil {
		return "", err
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	sh := os.Getenv("SHELL")
	if sh == "" {
		sh = "/bin/sh"
	}
	cmd := exec.CommandContext(runCtx, sh, "-c", command)
	cmd.Dir = t.Perm.Workspace
	cmd.Env = security.SafeSubprocessEnv(nil, t.Cfg.Tools.Shell.EnvBlacklistCompiled)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	return formatShellOutput(stdout.String(), stderr.String(), runErr)
}

func formatShellOutput(stdout, stderr string, runErr error) (string, error) {
	var b strings.Builder
	if stdout != "" {
		b.WriteString("stdout:\n")
		b.WriteString(stdout)
	}
	if stderr != "" {
		b.WriteString("stderr:\n")
		b.WriteString(stderr)
	}
	if runErr != nil {
		b.WriteString(ResultExitPrefix)
		b.WriteString(runErr.Error())
	}
	if b.Len() == 0 {
		return ResultNoOutput, nil
	}
	return b.String(), nil
}

var _ tool.Tool = (*ShellTool)(nil)
