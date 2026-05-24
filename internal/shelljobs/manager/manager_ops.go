package manager

import (
	"context"
	"fmt"
	"github.com/wzhejunqiu/ds-code/internal/shelljobs"
	"os"
	"sort"
	"time"
)

// Get returns job metadata and recent output.
func (m *Manager) Get(jobID string, maxBytes int) (shelljobs.Job, string, string, error) {
	job, err := m.readMeta(jobID)
	if err != nil {
		return shelljobs.Job{}, "", "", err
	}
	stdout := readTail(job.StdoutPath, maxBytes)
	stderr := readTail(job.StderrPath, maxBytes)
	return job, stdout, stderr, nil
}

// List returns all known jobs sorted by start time.
func (m *Manager) List() ([]shelljobs.Job, error) {
	entries, err := os.ReadDir(m.jobsDir)
	if err != nil {
		return nil, err
	}
	var jobs []shelljobs.Job
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
func (m *Manager) Cancel(ctx context.Context, jobID string) (shelljobs.Job, error) {
	_ = ctx
	m.mu.Lock()
	rj, ok := m.jobs[jobID]
	if ok && rj.cmd != nil && rj.cmd.Process != nil {
		_ = rj.cmd.Process.Kill()
		now := time.Now().UTC()
		rj.Status = shelljobs.StatusKilled
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
		return shelljobs.Job{}, err
	}
	if job.Status != shelljobs.StatusRunning || job.PID <= 0 {
		return job, fmt.Errorf("job %s is not running", jobID)
	}
	p, err := os.FindProcess(job.PID)
	if err != nil {
		return shelljobs.Job{}, err
	}
	_ = p.Kill()
	now := time.Now().UTC()
	job.Status = shelljobs.StatusKilled
	job.FinishedAt = &now
	code := -1
	job.ExitCode = &code
	_ = m.writeMeta(job)
	return job, nil
}
