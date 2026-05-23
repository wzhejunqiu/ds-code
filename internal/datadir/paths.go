// Fixed ~/.ds-code layout and per-project data paths (see README.md § Project identity).
package datadir

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const userDataDirName = ".ds-code"

// UserDataHome returns ~/.ds-code (fixed; not configurable).
func UserDataHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("datadir: user home: %w", err)
	}
	return filepath.Join(home, userDataDirName), nil
}

// UserConfigPath returns ~/.ds-code/config/config.yaml.
func UserConfigPath() (string, error) {
	root, err := UserDataHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "config", "config.yaml"), nil
}

// ProjectID returns hex(SHA256(projectRoot)).
func ProjectID(projectRoot string) string {
	sum := sha256.Sum256([]byte(projectRoot))
	return hex.EncodeToString(sum[:])
}

// ProjectDataDir returns ~/.ds-code/projects/<project_id>/.
func ProjectDataDir(projectRoot string) (string, error) {
	root, err := UserDataHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "projects", ProjectID(projectRoot)), nil
}

// DefaultDBPath returns the fixed sessions.db path for a project.
func DefaultDBPath(projectRoot string) string {
	dir, err := ProjectDataDir(projectRoot)
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "sessions.db")
}

// DefaultAuditLogPath returns the fixed audit.jsonl path for a project.
func DefaultAuditLogPath(projectRoot string) string {
	dir, err := ProjectDataDir(projectRoot)
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "audit.jsonl")
}

// DefaultShellJobsDir returns the fixed shell-jobs directory for a project.
func DefaultShellJobsDir(projectRoot string) string {
	dir, err := ProjectDataDir(projectRoot)
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "shell-jobs")
}

// DefaultCheckpointDir returns the fixed checkpoints directory for a project.
func DefaultCheckpointDir(projectRoot string) string {
	dir, err := ProjectDataDir(projectRoot)
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "checkpoints")
}

// DefaultLogsDir returns ~/.ds-code/projects/<project_id>/logs/.
func DefaultLogsDir(projectRoot string) string {
	dir, err := ProjectDataDir(projectRoot)
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "logs")
}

// DefaultLogPath returns the fixed application log file for a project.
func DefaultLogPath(projectRoot string) string {
	dir := DefaultLogsDir(projectRoot)
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "ds-code.log")
}

// EnsureLogsDir creates ~/.ds-code/projects/<id>/logs/ with mode 0700.
func EnsureLogsDir(projectRoot string) (string, error) {
	if _, err := EnsureProjectDataDir(projectRoot); err != nil {
		return "", err
	}
	dir := DefaultLogsDir(projectRoot)
	if dir == "" {
		return "", fmt.Errorf("datadir: logs dir for %q", projectRoot)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("datadir: create logs dir: %w", err)
	}
	return dir, nil
}

// EnsureProjectDataDir creates ~/.ds-code/projects/<id>/ with mode 0700
// and writes project.meta.json when missing.
func EnsureProjectDataDir(projectRoot string) (string, error) {
	dir, err := ProjectDataDir(projectRoot)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("datadir: create project data dir: %w", err)
	}
	metaPath := filepath.Join(dir, "project.meta.json")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		meta := struct {
			Root      string `json:"root"`
			CreatedAt string `json:"created_at"`
		}{
			Root:      projectRoot,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}
		b, err := json.MarshalIndent(meta, "", "  ")
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(metaPath, b, 0o600); err != nil {
			return "", fmt.Errorf("datadir: write project.meta.json: %w", err)
		}
	}
	return dir, nil
}

// EnsureUserConfigDir creates ~/.ds-code/config with mode 0700.
func EnsureUserConfigDir() error {
	root, err := UserDataHome()
	if err != nil {
		return err
	}
	return os.MkdirAll(filepath.Join(root, "config"), 0o700)
}
