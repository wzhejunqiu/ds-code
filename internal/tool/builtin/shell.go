package builtin

import (
	"context"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hejunqiu/ds-code/internal/config"
	"github.com/hejunqiu/ds-code/internal/permission"
	"github.com/hejunqiu/ds-code/internal/security"
	"github.com/hejunqiu/ds-code/internal/shelljobs"
	"github.com/hejunqiu/ds-code/internal/shelljobs/manager"
	"github.com/hejunqiu/ds-code/internal/tool"
)

// ShellTool runs shell commands synchronously or in the background.
type ShellTool struct {
	Cfg    *config.Config
	Perm   *permission.Engine
	Jobs   *manager.Manager
	Strict bool
}

func (t *ShellTool) Name() string { return "shell" }

func (t *ShellTool) Description() string {
	return "Run a shell command in the project workspace. Use background=true to start a job and job_id to poll or cancel."
}

func (t *ShellTool) Schema() map[string]any {
	return tool.ObjectSchema(map[string]any{
		"command": map[string]any{
			"type":        "string",
			"description": "Shell command (required for sync/background start)",
		},
		"background": map[string]any{
			"type":        "boolean",
			"description": "If true, start command in background and return job_id",
		},
		"job_id": map[string]any{
			"type":        "string",
			"description": "Poll output/status of a background job, or cancel when cancel=true",
		},
		"cancel": map[string]any{
			"type":        "boolean",
			"description": "Kill a background job (requires job_id)",
		},
		"list_jobs": map[string]any{
			"type":        "boolean",
			"description": "List background jobs for this project",
		},
	}, nil, t.Strict)
}

func (t *ShellTool) PermissionLevel() permission.Level { return permission.LevelHighest }

func (t *ShellTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var in struct {
		Command    string `json:"command"`
		Background bool   `json:"background"`
		JobID      string `json:"job_id"`
		Cancel     bool   `json:"cancel"`
		ListJobs   bool   `json:"list_jobs"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", err
	}

	if t.Jobs == nil {
		if in.Background || in.JobID != "" || in.ListJobs {
			return "", fmt.Errorf("shell background jobs are unavailable")
		}
		return t.runSync(ctx, in.Command)
	}

	if in.ListJobs {
		return t.formatList()
	}
	if in.JobID != "" {
		if in.Cancel {
			job, err := t.Jobs.Cancel(ctx, in.JobID)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("killed background job %s (%s)", job.ID, job.Command), nil
		}
		return t.formatJobStatus(in.JobID)
	}
	if in.Background {
		return t.startBackground(in.Command)
	}
	if strings.TrimSpace(in.Command) == "" {
		return "", fmt.Errorf("command is required (or use job_id / list_jobs)")
	}
	return t.runSync(ctx, in.Command)
}

func (t *ShellTool) startBackground(command string) (string, error) {
	job, err := t.Jobs.Start(command)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("background job started\njob_id: %s\npid: %d\nstatus: %s\ncommand: %s",
		job.ID, job.PID, job.Status, job.Command), nil
}

func (t *ShellTool) formatJobStatus(jobID string) (string, error) {
	maxBytes := t.Cfg.Tools.Shell.BackgroundOutputMaxBytes
	if maxBytes <= 0 {
		maxBytes = 256 * 1024
	}
	job, stdout, stderr, err := t.Jobs.Get(jobID, maxBytes)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "job_id: %s\nstatus: %s\ncommand: %s\npid: %d\nstarted: %s\n",
		job.ID, job.Status, job.Command, job.PID, job.StartedAt.Format(time.RFC3339))
	if job.FinishedAt != nil {
		fmt.Fprintf(&b, "finished: %s\n", job.FinishedAt.Format(time.RFC3339))
	}
	if job.ExitCode != nil {
		fmt.Fprintf(&b, "exit_code: %d\n", *job.ExitCode)
	}
	if stdout != "" {
		b.WriteString("\nstdout:\n")
		b.WriteString(stdout)
	}
	if stderr != "" {
		b.WriteString("\nstderr:\n")
		b.WriteString(stderr)
	}
	if stdout == "" && stderr == "" && job.Status == shelljobs.StatusRunning {
		b.WriteString("\n(no output yet)\n")
	}
	return strings.TrimSpace(b.String()), nil
}

func (t *ShellTool) formatList() (string, error) {
	jobs, err := t.Jobs.List()
	if err != nil {
		return "", err
	}
	if len(jobs) == 0 {
		return "No background shell jobs.", nil
	}
	var b strings.Builder
	b.WriteString("Background shell jobs:\n")
	for _, j := range jobs {
		exit := ""
		if j.ExitCode != nil {
			exit = fmt.Sprintf(" exit=%d", *j.ExitCode)
		}
		fmt.Fprintf(&b, "  %s  %s  pid=%d%s  %q\n",
			j.ID, j.Status, j.PID, exit, j.Command)
	}
	return b.String(), nil
}

func (t *ShellTool) runSync(ctx context.Context, command string) (string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
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
	cmd := exec.CommandContext(runCtx, shell, "-c", command)
	cmd.Dir = t.Perm.Workspace
	cmd.Env = security.SafeSubprocessEnv(nil, t.Cfg.Tools.Shell.EnvBlacklistCompiled)
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
