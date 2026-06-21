package manager

import (
	"context"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/shelljobs"
)

// Wait blocks until jobID finishes or ctx is canceled. On cancel, the job is killed.
func (m *Manager) Wait(ctx context.Context, jobID string) (shelljobs.Job, error) {
	if err := ctx.Err(); err != nil {
		return shelljobs.Job{}, err
	}

	var done <-chan struct{}
	m.mu.Lock()
	if rj, ok := m.jobs[jobID]; ok && rj.done != nil {
		done = rj.done
	}
	m.mu.Unlock()

	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			_, _ = m.Cancel(context.Background(), jobID)
			return shelljobs.Job{}, ctx.Err()
		}
		return m.readMeta(jobID)
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			_, _ = m.Cancel(context.Background(), jobID)
			return shelljobs.Job{}, ctx.Err()
		case <-ticker.C:
			job, err := m.readMeta(jobID)
			if err != nil {
				return shelljobs.Job{}, err
			}
			if job.Status != shelljobs.StatusRunning {
				return job, nil
			}
		}
	}
}
