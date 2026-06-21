package manager

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/config"
	"github.com/wzhejunqiu/ds-code/internal/shelljobs"
)

// Manager tracks background shell processes for one project.
type Manager struct {
	workspace string
	jobsDir   string
	cfg       config.ShellToolConfig

	mu          sync.Mutex
	jobs        map[string]*runningJob
	reconcileWG sync.WaitGroup
}

type runningJob struct {
	shelljobs.Job
	cmd  *exec.Cmd
	done chan struct{}
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
	m.startReconcileStaleJobs()
	return m, nil
}

func (m *Manager) startReconcileStaleJobs() {
	m.reconcileWG.Add(1)
	go func() {
		defer m.reconcileWG.Done()
		m.reconcileStaleJobs()
	}()
}

var reconcileMu sync.Mutex

func processAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

// reconcileStaleJobs fixes disk meta for jobs left running from a prior session.
// Orphan PIDs are killed; jobs are not re-attached to this manager.
func (m *Manager) reconcileStaleJobs() {
	reconcileMu.Lock()
	defer reconcileMu.Unlock()

	entries, err := os.ReadDir(m.jobsDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		if isTracked(m.jobsDir, id) {
			continue
		}
		job, err := m.readMeta(id)
		if err != nil || job.Status != shelljobs.StatusRunning {
			continue
		}
		now := time.Now().UTC()
		code := -1
		if job.PID > 0 && processAlive(job.PID) {
			_ = syscall.Kill(job.PID, syscall.SIGKILL)
			job.Status = shelljobs.StatusKilled
		} else {
			job.Status = shelljobs.StatusFailed
		}
		job.FinishedAt = &now
		job.ExitCode = &code
		_ = m.writeMeta(job)
	}
}

// Close kills all running jobs started in this session and releases resources.
func (m *Manager) Close() {
	m.reconcileWG.Wait()
	m.mu.Lock()
	var ids []string
	for id, rj := range m.jobs {
		if rj.Status == shelljobs.StatusRunning {
			ids = append(ids, id)
		}
	}
	m.mu.Unlock()
	for _, id := range ids {
		_, _ = m.Cancel(context.Background(), id)
	}
	m.mu.Lock()
	m.jobs = nil
	m.mu.Unlock()
}
