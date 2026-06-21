package manager

import (
	"fmt"
	"github.com/wzhejunqiu/ds-code/internal/security"
	"github.com/wzhejunqiu/ds-code/internal/shelljobs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Start launches a background shell command in workspace.
func (m *Manager) Start(command, description string) (shelljobs.Job, error) {
	command = strings.TrimSpace(command)
	description = strings.TrimSpace(description)
	if command == "" {
		return shelljobs.Job{}, fmt.Errorf("command is required")
	}
	maxBg := m.cfg.MaxBackground
	if maxBg <= 0 {
		maxBg = 5
	}

	m.mu.Lock()
	running := 0
	for _, j := range m.jobs {
		if j.Status == shelljobs.StatusRunning {
			running++
		}
	}
	if running >= maxBg {
		m.mu.Unlock()
		return shelljobs.Job{}, fmt.Errorf("shell: max background jobs (%d) reached", maxBg)
	}

	id := uuid.NewString()[:8]
	placeholder := &runningJob{
		Job: shelljobs.Job{
			ID:          id,
			Description: description,
			Command:     command,
			Status:      shelljobs.StatusRunning,
			StartedAt:   time.Now().UTC(),
		},
	}
	m.jobs[id] = placeholder
	m.mu.Unlock()

	jobDir := filepath.Join(m.jobsDir, id)
	if err := os.MkdirAll(jobDir, 0o700); err != nil {
		m.removeJob(id)
		return shelljobs.Job{}, err
	}
	stdoutPath := filepath.Join(jobDir, "stdout.log")
	stderrPath := filepath.Join(jobDir, "stderr.log")
	stdoutFile, err := os.Create(stdoutPath)
	if err != nil {
		m.removeJob(id)
		return shelljobs.Job{}, err
	}
	stderrFile, err := os.Create(stderrPath)
	if err != nil {
		_ = stdoutFile.Close()
		m.removeJob(id)
		return shelljobs.Job{}, err
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	cmd := exec.Command(shell, "-c", command)
	cmd.Dir = m.workspace
	cmd.Env = security.SafeSubprocessEnv(nil, m.cfg.EnvBlacklistCompiled)
	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile

	if err := cmd.Start(); err != nil {
		_ = stdoutFile.Close()
		_ = stderrFile.Close()
		m.removeJob(id)
		return shelljobs.Job{}, err
	}
	_ = stdoutFile.Close()
	_ = stderrFile.Close()

	job := shelljobs.Job{
		ID:          id,
		Description: description,
		Command:     command,
		PID:         cmd.Process.Pid,
		Status:      shelljobs.StatusRunning,
		StartedAt:   time.Now().UTC(),
		StdoutPath:  stdoutPath,
		StderrPath:  stderrPath,
	}
	if err := m.writeMeta(job); err != nil {
		_ = cmd.Process.Kill()
		m.removeJob(id)
		return shelljobs.Job{}, err
	}

	done := make(chan struct{})
	m.mu.Lock()
	rj := &runningJob{Job: job, cmd: cmd, done: done}
	m.jobs[id] = rj
	m.mu.Unlock()

	go m.waitJob(id, cmd, done)
	trackJob(m.jobsDir, id)
	return job, nil
}

func (m *Manager) removeJob(id string) {
	m.mu.Lock()
	delete(m.jobs, id)
	m.mu.Unlock()
	untrackJob(m.jobsDir, id)
}

func (m *Manager) waitJob(id string, cmd *exec.Cmd, done chan struct{}) {
	defer close(done)
	defer untrackJob(m.jobsDir, id)
	err := cmd.Wait()
	code := 0
	status := shelljobs.StatusCompleted
	if err != nil {
		status = shelljobs.StatusFailed
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			code = -1
		}
	}
	now := time.Now().UTC()

	m.mu.Lock()
	defer m.mu.Unlock()
	rj, ok := m.jobs[id]
	if !ok {
		return
	}
	if rj.Status == shelljobs.StatusKilled {
		return
	}
	rj.Status = status
	rj.ExitCode = &code
	rj.FinishedAt = &now
	rj.cmd = nil
	_ = m.writeMeta(rj.Job)
}
