package shell

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

func (t *ShellTool) Description() string { return DescShell }

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
		"background": map[string]any{
			"type":        "boolean",
			"description": SchemaBackground,
		},
		"job_id": map[string]any{
			"type":        "string",
			"description": SchemaJobID,
		},
		"cancel": map[string]any{
			"type":        "boolean",
			"description": SchemaCancel,
		},
		"list_jobs": map[string]any{
			"type":        "boolean",
			"description": SchemaListJobs,
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
			return "", fmt.Errorf("%s", ErrBackgroundUnavailable)
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
			return fmt.Sprintf(ResultKilledJob, job.ID, job.Command), nil
		}
		return t.formatJobStatus(in.JobID)
	}
	if in.Background {
		return t.startBackground(in.Command)
	}
	if strings.TrimSpace(in.Command) == "" {
		return "", fmt.Errorf("%s", ErrCommandRequired)
	}
	return t.runSync(ctx, in.Command)
}

func (t *ShellTool) startBackground(command string) (string, error) {
	job, err := t.Jobs.Start(command)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(ResultBackgroundStarted,
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
	fmt.Fprintf(&b, ResultJobHeader,
		job.ID, job.Status, job.Command, job.PID, job.StartedAt.Format(time.RFC3339))
	if job.FinishedAt != nil {
		fmt.Fprintf(&b, ResultJobFinished, job.FinishedAt.Format(time.RFC3339))
	}
	if job.ExitCode != nil {
		fmt.Fprintf(&b, ResultJobExitCode, *job.ExitCode)
	}
	if stdout != "" {
		b.WriteString(ResultStdout)
		b.WriteString(stdout)
	}
	if stderr != "" {
		b.WriteString(ResultStderr)
		b.WriteString(stderr)
	}
	if stdout == "" && stderr == "" && job.Status == shelljobs.StatusRunning {
		b.WriteString(ResultNoOutputYet)
	}
	return strings.TrimSpace(b.String()), nil
}

func (t *ShellTool) formatList() (string, error) {
	jobs, err := t.Jobs.List()
	if err != nil {
		return "", err
	}
	if len(jobs) == 0 {
		return ResultNoBackgroundJobs, nil
	}
	var b strings.Builder
	b.WriteString(ResultJobListHeader)
	for _, j := range jobs {
		exit := ""
		if j.ExitCode != nil {
			exit = fmt.Sprintf(" exit=%d", *j.ExitCode)
		}
		fmt.Fprintf(&b, ResultJobListLine,
			j.ID, j.Status, j.PID, exit, j.Command)
	}
	return b.String(), nil
}

func (t *ShellTool) runSync(ctx context.Context, command string) (string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", fmt.Errorf("%s", ErrCommandRequiredSync)
	}

	timeout := t.Cfg.Tools.Shell.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
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
		b.WriteString(ResultExitPrefix)
		b.WriteString(err.Error())
	}
	if b.Len() == 0 {
		return ResultNoOutput, nil
	}
	return b.String(), nil
}

var _ tool.Tool = (*ShellTool)(nil)
