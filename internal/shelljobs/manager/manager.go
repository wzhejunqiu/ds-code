package manager

import (
	"fmt"
	"github.com/hejunqiu/ds-code/internal/shelljobs"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

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
	shelljobs.Job
	cmd *exec.Cmd
}

// OpenManager creates a shell job manager for a project.
func Open(projectRoot string, cfg config.ShellToolConfig) (*Manager, error) {
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
		if job.Status == shelljobs.StatusRunning {
			// Process may have died while ds-code was down; verify PID.
			if job.PID > 0 && processAlive(job.PID) {
				m.jobs[id] = &runningJob{Job: job}
			} else {
				job.Status = shelljobs.StatusFailed
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
