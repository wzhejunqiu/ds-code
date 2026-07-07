// Package datadir provides desktop-isolated data paths under ~/.ds-code/desktop/projects/.
package datadir

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wzhejunqiu/ds-code/internal/datadir"
)

// ProjectID returns hex(SHA256(projectRoot)), same algorithm as CLI.
func ProjectID(projectRoot string) string {
	return datadir.ProjectID(projectRoot)
}

// ProjectDataDir returns ~/.ds-code/desktop/projects/<project_id>/.
func ProjectDataDir(projectRoot string) (string, error) {
	root, err := datadir.UserDataHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "desktop", "projects", ProjectID(projectRoot)), nil
}

// DefaultDBPath returns the desktop sessions.db path for a project.
func DefaultDBPath(projectRoot string) string {
	dir, err := ProjectDataDir(projectRoot)
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "sessions.db")
}

// DefaultAuditLogPath returns the desktop audit.jsonl path for a project.
func DefaultAuditLogPath(projectRoot string) string {
	dir, err := ProjectDataDir(projectRoot)
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "audit.jsonl")
}

// DefaultCheckpointDir returns the desktop checkpoints directory for a project.
func DefaultCheckpointDir(projectRoot string) string {
	dir, err := ProjectDataDir(projectRoot)
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "checkpoints")
}

// DefaultShellJobsDir returns the desktop shell-jobs directory for a project.
func DefaultShellJobsDir(projectRoot string) string {
	dir, err := ProjectDataDir(projectRoot)
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "shell-jobs")
}

// EnsureProjectDataDir creates ~/.ds-code/desktop/projects/<id>/ with mode 0700
// and writes project.meta.json when missing.
func EnsureProjectDataDir(projectRoot string) (string, error) {
	dir, err := ProjectDataDir(projectRoot)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("desktop datadir: create project data dir: %w", err)
	}
	metaPath := filepath.Join(dir, "project.meta.json")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		meta := struct {
			Root      string `json:"root"`
			CreatedAt string `json:"created_at"`
			Runtime   string `json:"runtime"`
		}{
			Root:      projectRoot,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			Runtime:   "desktop",
		}
		b, err := json.MarshalIndent(meta, "", "  ")
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(metaPath, b, 0o600); err != nil {
			return "", fmt.Errorf("desktop datadir: write project.meta.json: %w", err)
		}
	}
	return dir, nil
}
