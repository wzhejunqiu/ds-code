package permission

import (
	"fmt"
	"path/filepath"

	wspkg "github.com/hejunqiu/ds-code/internal/workspace"
)

// EnsureAbsUnderWorkspace reports whether abs lies under workspace.
func (e *Engine) EnsureAbsUnderWorkspace(abs, original string) error {
	if err := wspkg.EnsureAbsUnder(e.Workspace, abs); err != nil {
		return fmt.Errorf("%w: outside workspace: %s", ErrDenied, original)
	}
	return nil
}

// ResolveRelUnderWorkspace joins rel to workspace and ensures the result is inside workspace.
func (e *Engine) ResolveRelUnderWorkspace(rel string) (string, error) {
	abs, err := wspkg.ResolveRel(e.Workspace, rel)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDenied, err)
	}
	return abs, nil
}

// ValidateAbsPathsUnderWorkspace checks each absolute path; fails on first path outside workspace.
func (e *Engine) ValidateAbsPathsUnderWorkspace(paths []string, contextLabel string) error {
	for _, p := range paths {
		abs := filepath.Clean(p)
		if err := e.EnsureAbsUnderWorkspace(abs, p); err != nil {
			if contextLabel != "" {
				return fmt.Errorf("%s: %w", contextLabel, err)
			}
			return err
		}
	}
	return nil
}

// GlobOutsideWorkspaceError formats a glob escape error.
func GlobOutsideWorkspaceError(abs, pattern string) error {
	return fmt.Errorf("glob: path outside workspace: %s (pattern: %q)", abs, pattern)
}

// PatchOutsideWorkspaceError formats a patch path escape error.
func PatchOutsideWorkspaceError(rel string) error {
	return fmt.Errorf("patch: path outside workspace: %s", rel)
}

// IsUnderWorkspace reports whether abs is inside workspace without an Engine (for tests).
func IsUnderWorkspace(workspace, abs string) bool {
	return wspkg.EnsureAbsUnder(workspace, abs) == nil
}
