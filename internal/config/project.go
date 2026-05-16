package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveProjectRoot walks from startDir upward for a git root; otherwise returns
// the cleaned absolute path of startDir (with symlinks evaluated when possible).
func ResolveProjectRoot(startDir string) (string, error) {
	abs, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("config: project root: %w", err)
	}
	abs = filepath.Clean(abs)

	dir := abs
	for {
		if isGitRoot(dir) {
			if resolved, err := filepath.EvalSymlinks(dir); err == nil {
				return filepath.Clean(resolved), nil
			}
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return filepath.Clean(resolved), nil
	}
	return abs, nil
}

func isGitRoot(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir() || !info.IsDir() // file .git (worktree) or directory
}

// ProjectConfigPath returns <git-root>/.ds-code/config.yaml if the file exists.
func ProjectConfigPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".ds-code", "config.yaml")
}
