// Fixed ~/.ds-code layout and per-project data paths (see README.md § Project identity).
package config

import "github.com/wzhejunqiu/ds-code/internal/datadir"

// UserDataHome returns ~/.ds-code (fixed; not configurable).
func UserDataHome() (string, error) {
	return datadir.UserDataHome()
}

// UserConfigPath returns ~/.ds-code/config/config.yaml.
func UserConfigPath() (string, error) {
	return datadir.UserConfigPath()
}

// ProjectID returns hex(SHA256(projectRoot)).
func ProjectID(projectRoot string) string {
	return datadir.ProjectID(projectRoot)
}

// ProjectDataDir returns ~/.ds-code/projects/<project_id>/.
func ProjectDataDir(projectRoot string) (string, error) {
	return datadir.ProjectDataDir(projectRoot)
}

// DefaultDBPath returns the fixed sessions.db path for a project.
func DefaultDBPath(projectRoot string) string {
	return datadir.DefaultDBPath(projectRoot)
}

// DefaultAuditLogPath returns the fixed audit.jsonl path for a project.
func DefaultAuditLogPath(projectRoot string) string {
	return datadir.DefaultAuditLogPath(projectRoot)
}

// DefaultShellJobsDir returns the fixed shell-jobs directory for a project.
func DefaultShellJobsDir(projectRoot string) string {
	return datadir.DefaultShellJobsDir(projectRoot)
}

// DefaultCheckpointDir returns the fixed checkpoints directory for a project.
func DefaultCheckpointDir(projectRoot string) string {
	return datadir.DefaultCheckpointDir(projectRoot)
}

// DefaultLogsDir returns ~/.ds-code/projects/<project_id>/logs/.
func DefaultLogsDir(projectRoot string) string {
	return datadir.DefaultLogsDir(projectRoot)
}

// DefaultLogPath returns the fixed application log file for a project.
func DefaultLogPath(projectRoot string) string {
	return datadir.DefaultLogPath(projectRoot)
}

// EnsureLogsDir creates ~/.ds-code/projects/<id>/logs/ with mode 0700.
func EnsureLogsDir(projectRoot string) (string, error) {
	return datadir.EnsureLogsDir(projectRoot)
}

// EnsureProjectDataDir creates ~/.ds-code/projects/<id>/ with mode 0700
// and writes project.meta.json when missing.
func EnsureProjectDataDir(projectRoot string) (string, error) {
	return datadir.EnsureProjectDataDir(projectRoot)
}

// EnsureUserConfigDir creates ~/.ds-code/config with mode 0700.
func EnsureUserConfigDir() error {
	return datadir.EnsureUserConfigDir()
}
