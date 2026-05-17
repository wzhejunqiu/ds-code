package manager

import (
	"encoding/json"
	"fmt"
	"github.com/hejunqiu/ds-code/internal/shelljobs"
	"os"
	"path/filepath"
)

func (m *Manager) metaPath(id string) string {
	return filepath.Join(m.jobsDir, id, "meta.json")
}

func (m *Manager) writeMeta(job shelljobs.Job) error {
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

func (m *Manager) readMeta(id string) (shelljobs.Job, error) {
	b, err := os.ReadFile(m.metaPath(id))
	if err != nil {
		return shelljobs.Job{}, err
	}
	var job shelljobs.Job
	if err := json.Unmarshal(b, &job); err != nil {
		return shelljobs.Job{}, err
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
