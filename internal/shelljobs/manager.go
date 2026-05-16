package shelljobs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/hejunqiu/ds-code/internal/config"
)

// Manager tracks background shell processes for one project.
type Manager struct {
	workspace string
	jobsDir   string
	cfg       config.ShellToolConfig

	mu   sync.Mutex
	jobs map[string]*runningJob
}

type runningJob struct {
	Job
	cmd *exec.Cmd
}

// OpenManager creates a shell job manager for a project.
func OpenManager(projectRoot string, cfg config.ShellToolConfig) (*Manager, error) {
	if _, err := config.EnsureProjectDataDir(projectRoot); err != nil {
		return nil, err
	}
	dir := config.DefaultShellJobsDir(projectRoot)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("shelljobs: mkdir: %w", err)
	}
	m := &Manager{
		workspace: projectRoot,
		jobsDir:   dir,
		cfg:       cfg,
		jobs:      make(map[string]*runningJob),
	}
	m.loadExisting()
	return m, nil
}

func (m *Manager) loadExisting() {
	entries, err := os.ReadDir(m.jobsDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		job, err := m.readMeta(id)
		if err != nil {
			continue
		}
		if job.Status == StatusRunning {
			// Process may have died while ds-code was down; verify PID.
			if job.PID > 0 && processAlive(job.PID) {
				m.jobs[id] = &runningJob{Job: job}
			} else {
				job.Status = StatusFailed
				now := time.Now().UTC()
				job.FinishedAt = &now
				code := -1
				job.ExitCode = &code
				_ = m.writeMeta(job)
			}
		}
	}
}

func processAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

// Start launches a background shell command in workspace.
func (m *Manager) Start(command string) (Job, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return Job{}, fmt.Errorf("command is required")
	}
	maxBg := m.cfg.MaxBackground
	if maxBg <= 0 {
		maxBg = 5
	}
	m.mu.Lock()
	running := 0
	for _, j := range m.jobs {
		if j.Status == StatusRunning {
			running++
		}
	}
	m.mu.Unlock()
	if running >= maxBg {
		return Job{}, fmt.Errorf("shell: max background jobs (%d) reached", maxBg)
	}

	id := uuid.NewString()[:8]
	jobDir := filepath.Join(m.jobsDir, id)
	if err := os.MkdirAll(jobDir, 0o700); err != nil {
		return Job{}, err
	}
	stdoutPath := filepath.Join(jobDir, "stdout.log")
	stderrPath := filepath.Join(jobDir, "stderr.log")
	stdoutFile, err := os.Create(stdoutPath)
	if err != nil {
		return Job{}, err
	}
	stderrFile, err := os.Create(stderrPath)
	if err != nil {
		_ = stdoutFile.Close()
		return Job{}, err
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	cmd := exec.Command(shell, "-c", command)
	cmd.Dir = m.workspace
	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile

	if err := cmd.Start(); err != nil {
		_ = stdoutFile.Close()
		_ = stderrFile.Close()
		return Job{}, err
	}
	_ = stdoutFile.Close()
	_ = stderrFile.Close()

	job := Job{
		ID:         id,
		Command:    command,
		PID:        cmd.Process.Pid,
		Status:     StatusRunning,
		StartedAt:  time.Now().UTC(),
		StdoutPath: stdoutPath,
		StderrPath: stderrPath,
	}
	if err := m.writeMeta(job); err != nil {
		_ = cmd.Process.Kill()
		return Job{}, err
	}

	rj := &runningJob{Job: job, cmd: cmd}
	m.mu.Lock()
	m.jobs[id] = rj
	m.mu.Unlock()

	go m.waitJob(id, cmd)
	return job, nil
}

func (m *Manager) waitJob(id string, cmd *exec.Cmd) {
	err := cmd.Wait()
	code := 0
	status := StatusCompleted
	if err != nil {
		status = StatusFailed
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
	if rj.Status == StatusKilled {
		return
	}
	rj.Status = status
	rj.ExitCode = &code
	rj.FinishedAt = &now
	rj.cmd = nil
	_ = m.writeMeta(rj.Job)
}

// Get returns job metadata and recent output.
func (m *Manager) Get(jobID string, maxBytes int) (Job, string, string, error) {
	job, err := m.readMeta(jobID)
	if err != nil {
		return Job{}, "", "", err
	}
	stdout := readTail(job.StdoutPath, maxBytes)
	stderr := readTail(job.StderrPath, maxBytes)
	return job, stdout, stderr, nil
}

// List returns all known jobs sorted by start time.
func (m *Manager) List() ([]Job, error) {
	entries, err := os.ReadDir(m.jobsDir)
	if err != nil {
		return nil, err
	}
	var jobs []Job
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		job, err := m.readMeta(e.Name())
		if err != nil {
			continue
		}
		jobs = append(jobs, job)
	}
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].StartedAt.Before(jobs[j].StartedAt)
	})
	return jobs, nil
}

// Cancel kills a running background job.
func (m *Manager) Cancel(ctx context.Context, jobID string) (Job, error) {
	_ = ctx
	m.mu.Lock()
	rj, ok := m.jobs[jobID]
	if ok && rj.cmd != nil && rj.cmd.Process != nil {
		_ = rj.cmd.Process.Kill()
		now := time.Now().UTC()
		rj.Status = StatusKilled
		rj.FinishedAt = &now
		code := -1
		rj.ExitCode = &code
		rj.cmd = nil
		job := rj.Job
		m.mu.Unlock()
		_ = m.writeMeta(job)
		return job, nil
	}
	m.mu.Unlock()
	job, err := m.readMeta(jobID)
	if err != nil {
		return Job{}, err
	}
	if job.Status != StatusRunning || job.PID <= 0 {
		return job, fmt.Errorf("job %s is not running", jobID)
	}
	p, err := os.FindProcess(job.PID)
	if err != nil {
		return Job{}, err
	}
	_ = p.Kill()
	now := time.Now().UTC()
	job.Status = StatusKilled
	job.FinishedAt = &now
	code := -1
	job.ExitCode = &code
	_ = m.writeMeta(job)
	return job, nil
}

// Close kills all in-memory tracked running jobs.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, rj := range m.jobs {
		if rj.cmd != nil && rj.cmd.Process != nil {
			_ = rj.cmd.Process.Kill()
		}
	}
}

func (m *Manager) metaPath(id string) string {
	return filepath.Join(m.jobsDir, id, "meta.json")
}

func (m *Manager) writeMeta(job Job) error {
	dir := filepath.Join(m.jobsDir, job.ID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.metaPath(job.ID), b, 0o600)
}

func (m *Manager) readMeta(id string) (Job, error) {
	b, err := os.ReadFile(m.metaPath(id))
	if err != nil {
		return Job{}, err
	}
	var job Job
	if err := json.Unmarshal(b, &job); err != nil {
		return Job{}, err
	}
	return job, nil
}

func readTail(path string, maxBytes int) string {
	if maxBytes <= 0 {
		maxBytes = 256 * 1024
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		return fmt.Sprintf("(read error: %v)", err)
	}
	if len(b) <= maxBytes {
		return string(b)
	}
	return "...(truncated)\n" + string(b[len(b)-maxBytes:])
}
