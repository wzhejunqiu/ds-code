package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveRel resolves rel under workspace, evaluating symlinks, and ensures the path stays inside workspace.
func ResolveRel(workspace, rel string) (string, error) {
	if workspace == "" {
		return "", fmt.Errorf("workspace: empty workspace")
	}
	ws, err := evalWorkspaceRoot(workspace)
	if err != nil {
		return "", err
	}

	if filepath.IsAbs(rel) {
		abs := filepath.Clean(rel)
		if resolved, err := filepath.EvalSymlinks(abs); err == nil {
			abs = resolved
		}
		if err := ensureUnder(ws, abs); err != nil {
			return "", err
		}
		return abs, nil
	}

	if strings.Contains(rel, "..") {
		return "", fmt.Errorf("workspace: path traversal: %s", rel)
	}

	abs := filepath.Join(ws, filepath.Clean(rel))
	abs, err = resolvePath(ws, abs)
	if err != nil {
		return "", err
	}
	if err := ensureUnder(ws, abs); err != nil {
		return "", fmt.Errorf("workspace: outside workspace: %s", rel)
	}
	return abs, nil
}

// ValidateRel checks that rel resolves inside workspace.
func ValidateRel(workspace, rel string) error {
	if workspace == "" {
		return nil
	}
	_, err := ResolveRel(workspace, rel)
	return err
}

// EnsureAbsUnder verifies abs lies inside workspace.
func EnsureAbsUnder(workspace, abs string) error {
	if workspace == "" {
		return nil
	}
	ws, err := evalWorkspaceRoot(workspace)
	if err != nil {
		return err
	}
	abs = filepath.Clean(abs)
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	} else if parent, err := filepath.EvalSymlinks(filepath.Dir(abs)); err == nil {
		abs = filepath.Join(parent, filepath.Base(abs))
	}
	return ensureUnder(ws, abs)
}

func evalWorkspaceRoot(workspace string) (string, error) {
	ws, err := filepath.EvalSymlinks(workspace)
	if err != nil {
		ws, err = filepath.Abs(workspace)
		if err != nil {
			return "", err
		}
	}
	return filepath.Clean(ws), nil
}

func resolvePath(ws, abs string) (string, error) {
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved, nil
	}
	if _, statErr := os.Stat(abs); statErr != nil {
		parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
		if err != nil {
			return "", fmt.Errorf("workspace: cannot resolve parent of %s", abs)
		}
		return filepath.Join(parent, filepath.Base(abs)), nil
	}
	return abs, nil
}

func ensureUnder(ws, abs string) error {
	relTo, err := filepath.Rel(ws, abs)
	if err != nil || strings.HasPrefix(relTo, "..") || relTo == ".." {
		return fmt.Errorf("workspace: path outside workspace: %s", abs)
	}
	return nil
}
